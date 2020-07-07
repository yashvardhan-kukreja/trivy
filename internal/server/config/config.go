package config

import (
	"github.com/aquasecurity/trivy/internal/config"
	"github.com/urfave/cli/v2"

	"github.com/prometheus/client_golang/prometheus"
)

type Config struct {
	config.GlobalConfig
	config.DBConfig

	Listen          string
	Token           string
	TokenHeader     string
	MetricsRegistry *prometheus.Registry
	GaugeMetric     *prometheus.GaugeVec
}

func New(c *cli.Context) Config {
	// the error is ignored because logger is unnecessary
	gc, _ := config.NewGlobalConfig(c)

	return Config{
		GlobalConfig: gc,
		DBConfig:     config.NewDBConfig(c),

		Listen:          c.String("listen"),
		Token:           c.String("token"),
		TokenHeader:     c.String("token-header"),
		MetricsRegistry: prometheus.NewRegistry(),
	}
}

func (c *Config) Init() (err error) {
	if err := c.DBConfig.Init(); err != nil {
		return err
	}
	if c.MetricsRegistry != nil {
		c.GaugeMetric = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "trivy",
				Help: "Gauge Metrics associated with trivy - Last DB Update, Last DB Update Attempt ...",
			},
			[]string{"action"},
		)
		c.MetricsRegistry.MustRegister(c.GaugeMetric)
	}
	return nil
}
