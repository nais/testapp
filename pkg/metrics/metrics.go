package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	namespace = "nais"
	subsystem = "testapp"
)

func gauge(name, help string) prometheus.Gauge {
	return prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      name,
		Help:      help,
		Namespace: namespace,
		Subsystem: subsystem,
	})
}

var (
	LeadTime            = gauge("lead_time", "Seconds used in deployment pipeline, from making the request until the application is available")
	TimeSinceLastDeploy = gauge("time_since_last_deploy", "Seconds since last deploy of this application")
)

func init() {
	prometheus.MustRegister(LeadTime)
	prometheus.MustRegister(TimeSinceLastDeploy)
}

func Handler() http.Handler {
	return promhttp.Handler()
}
