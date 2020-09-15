package extendedConfig

import (
	"github.com/aquasecurity/trivy/internal/server/config"
	"github.com/prometheus/client_golang/prometheus"
)

type ExtendedConfig struct {
	Config          config.Config
	MetricsRegistry *prometheus.Registry
	GaugeMetric     *prometheus.GaugeVec
}

func New(c config.Config) ExtendedConfig {
	return ExtendedConfig{
		Config: c,
	}
}

func (ec *ExtendedConfig) Init() {
	ec.MetricsRegistry = prometheus.NewRegistry()
	ec.GaugeMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "trivy",
			Help: "Gauge Metrics associated with trivy - Last DB Update, Last DB Update Attempt ...",
		},
		[]string{"action"},
	)
	ec.MetricsRegistry.MustRegister(ec.GaugeMetric)
}
