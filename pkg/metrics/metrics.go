package metrics

import (
	"net/http"
	"time"

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

//func histogram(name, help string) prometheus.Histogram {
//	return prometheus.NewHistogram(prometheus.HistogramOpts{
//		Namespace: namespace,
//		Subsystem: subsystem,
//		Name:      name,
//		Help:      help,
//		Buckets:   []float64{10000, 100000, 1000000, 2000000, 4000000, 8000000},
//	})
//}

var (
	LeadTime        = gauge("lead_time", "Seconds used in deployment pipeline, from making the request until the application is available")
	TimeSinceDeploy = gauge("time_since_deploy", "Seconds since the latest deploy of this application")

	DeployTimestamp = gauge("deploy_timestamp", "Timestamp when the deploy of this application was triggered in the pipeline")
	StartTimestamp  = gauge("start_timestamp", "Start time of the application")
	BucketWrite = gauge("bucket_write_latency", "The time it takes to write to the bucket in nanoseconds")
	BucketRead = gauge("bucket_read_latency", "The time it takes to read to the bucket in nanoseconds")
	DbInsert = gauge("db_insert", "The time it takes to insert to table")
	DbRead = gauge("db_read", "The time it takes to read from table")
)

func init() {
	prometheus.MustRegister(LeadTime)
	prometheus.MustRegister(TimeSinceDeploy)
	prometheus.MustRegister(DeployTimestamp)
	prometheus.MustRegister(StartTimestamp)
	prometheus.MustRegister(BucketWrite)
	prometheus.MustRegister(BucketRead)
	prometheus.MustRegister(DbInsert)
	prometheus.MustRegister(DbRead)
}

func SetLatencyMetric(start time.Time, gauge prometheus.Gauge) time.Duration {
	latency := time.Since(start)
	gauge.Set(float64(latency.Nanoseconds()))
	return latency
}

func Handler() http.Handler {
	return promhttp.Handler()
}
