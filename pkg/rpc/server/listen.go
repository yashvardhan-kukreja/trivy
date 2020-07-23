package server

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/afero"

	"github.com/google/wire"
	"github.com/twitchtv/twirp"
	"golang.org/x/xerrors"

	"github.com/aquasecurity/fanal/cache"
	"github.com/aquasecurity/trivy-db/pkg/db"
	"github.com/aquasecurity/trivy/internal/server/config"
	dbFile "github.com/aquasecurity/trivy/pkg/db"
	"github.com/aquasecurity/trivy/pkg/log"
	"github.com/aquasecurity/trivy/pkg/utils"
	rpcCache "github.com/aquasecurity/trivy/rpc/cache"
	"github.com/aquasecurity/trivy/rpc/detector"
	rpcDetector "github.com/aquasecurity/trivy/rpc/detector"
	rpcScanner "github.com/aquasecurity/trivy/rpc/scanner"
)

var DBWorkerSuperSet = wire.NewSet(
	dbFile.SuperSet,
	newDBWorker,
)

func ListenAndServe(c config.Config, fsCache cache.FSCache) error {
	requestWg := &sync.WaitGroup{}
	dbUpdateWg := &sync.WaitGroup{}

	withWaitGroup := func(base http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Stop processing requests during DB update
			dbUpdateWg.Wait()

			// Wait for all requests to be processed before DB update
			requestWg.Add(1)
			defer requestWg.Done()

			base.ServeHTTP(w, r)

		})
	}

	go func() {
		worker := initializeDBWorker(c.CacheDir, true)
		if err := initializeMetricGauge(c.GaugeMetric, c.CacheDir); err != nil {
			c.GaugeMetric = nil // just in case some if some error/issue occurs (probably with GaugeMetric). So, making c.GaugeMetric nil and hence, unusable for any of the functions which require it
			log.Logger.Errorf("%+v\n", err)
		}
		ctx := context.Background()
		for {
			time.Sleep(1 * time.Hour)
			if err := worker.update(ctx, c.AppVersion, c.CacheDir, dbUpdateWg, requestWg, c.GaugeMetric); err != nil {
				log.Logger.Errorf("%+v\n", err)
			}
		}
	}()

	mux := http.NewServeMux()

	scanHandler := rpcScanner.NewScannerServer(initializeScanServer(fsCache), nil)
	mux.Handle(rpcScanner.ScannerPathPrefix, withToken(withWaitGroup(scanHandler), c.Token, c.TokenHeader))

	layerHandler := rpcCache.NewCacheServer(NewCacheServer(fsCache), nil)
	mux.Handle(rpcCache.CachePathPrefix, withToken(withWaitGroup(layerHandler), c.Token, c.TokenHeader))

	// osHandler is for backward compatibility
	osHandler := rpcDetector.NewOSDetectorServer(initializeOspkgServer(), nil)
	mux.Handle(rpcDetector.OSDetectorPathPrefix, withToken(withWaitGroup(osHandler), c.Token, c.TokenHeader))

	// libHandler is for backward compatibility
	libHandler := rpcDetector.NewLibDetectorServer(initializeLibServer(), nil)
	mux.Handle(rpcDetector.LibDetectorPathPrefix, withToken(withWaitGroup(libHandler), c.Token, c.TokenHeader))

	// promHandler is for dealing with update the custom prometheus metrics
	promHandler := promhttp.HandlerFor(c.MetricsRegistry, promhttp.HandlerOpts{Timeout: 10 * time.Second})
	mux.Handle("/metrics", withToken(withWaitGroup(promHandler), c.Token, c.TokenHeader))

	log.Logger.Infof("Listening %s...", c.Listen)

	return http.ListenAndServe(c.Listen, mux)
}

func withToken(base http.Handler, token, tokenHeader string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token != "" && token != r.Header.Get(tokenHeader) {
			detector.WriteError(w, twirp.NewError(twirp.Unauthenticated, "invalid token"))
			return
		}
		base.ServeHTTP(w, r)
	})
}

type dbWorker struct {
	dbClient dbFile.Operation
}

func newDBWorker(dbClient dbFile.Operation) dbWorker {
	return dbWorker{dbClient: dbClient}
}

func (w dbWorker) update(ctx context.Context, appVersion, cacheDir string,
	dbUpdateWg, requestWg *sync.WaitGroup, gaugeMetric *prometheus.GaugeVec) error {
	if err := updateLastDBUpdatePrometheus(gaugeMetric, float64(time.Now().Unix()), true); err != nil { //updating the last_db_update_attempt prom metric coz a db update attempt got triggered
		return xerrors.Errorf("%w", err)
	}
	log.Logger.Debug("Check for DB update...")
	needsUpdate, err := w.dbClient.NeedsUpdate(appVersion, false, false)
	if err != nil {
		return xerrors.Errorf("failed to check if db needs an update")
	} else if !needsUpdate {
		return nil
	}

	log.Logger.Info("Updating DB...")
	if err = w.hotUpdate(ctx, cacheDir, dbUpdateWg, requestWg); err != nil {
		return xerrors.Errorf("failed DB hot update")
	}
	if err = updateLastDBUpdatePrometheus(gaugeMetric, float64(time.Now().Unix()), false); err != nil { //updating the last_db_update metric (here, coz code reaches till here, then it's a successful db update)
		return xerrors.Errorf("%w", err)
	}
	return nil
}

func (w dbWorker) hotUpdate(ctx context.Context, cacheDir string, dbUpdateWg, requestWg *sync.WaitGroup) error {
	tmpDir, err := ioutil.TempDir("", "db")
	if err != nil {
		return xerrors.Errorf("failed to create a temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := w.dbClient.Download(ctx, tmpDir, false); err != nil {
		return xerrors.Errorf("failed to download vulnerability DB: %w", err)
	}

	log.Logger.Info("Suspending all requests during DB update")
	dbUpdateWg.Add(1)
	defer dbUpdateWg.Done()

	log.Logger.Info("Waiting for all requests to be processed before DB update...")
	requestWg.Wait()

	if err = db.Close(); err != nil {
		return xerrors.Errorf("failed to close DB: %w", err)
	}

	if _, err = utils.CopyFile(db.Path(tmpDir), db.Path(cacheDir)); err != nil {
		return xerrors.Errorf("failed to copy the database file: %w", err)
	}

	if err = w.dbClient.UpdateMetadata(cacheDir); err != nil {
		return xerrors.Errorf("unable to update database metadata: %w", err)
	}

	log.Logger.Info("Reopening DB...")
	if err = db.Init(cacheDir); err != nil {
		return xerrors.Errorf("failed to open DB: %w", err)
	}

	return nil
}

func initializeMetricGauge(gauge *prometheus.GaugeVec, cacheDir string) error {
	// the nil gauge situation will be caught by updateLastDBUpdatePrometheus function
	m := dbFile.NewMetadata(afero.NewOsFs(), cacheDir)
	metadata, err := m.Get()
	if err != nil {
		return xerrors.Errorf("Error initialising the metrics for prometheus endpoint: %w", err)
	}
	if err = updateLastDBUpdatePrometheus(gauge, float64(metadata.UpdatedAt.Unix()), true); err != nil { //updating the last_db_update_attempt prom metric coz a db update attempt got triggered
		return xerrors.Errorf("%w", err)
	}
	if err = updateLastDBUpdatePrometheus(gauge, float64(metadata.UpdatedAt.Unix()), false); err != nil { //updating the last_db_update metric (here, coz code reaches till here, then it's a successful db update)
		return xerrors.Errorf("%w", err)
	}
	return nil
}

func updateLastDBUpdatePrometheus(gauge *prometheus.GaugeVec, time float64, onlyDBAttempt bool) error {
	if gauge == nil {
		return xerrors.Errorf("Prometheus gauge found to be nil")
	}
	if onlyDBAttempt {
		gauge.With(prometheus.Labels{"action": "last_db_update_attempt"}).Set(time)
	} else {
		gauge.With(prometheus.Labels{"action": "last_db_update"}).Set(time)
	}
	return nil
}
