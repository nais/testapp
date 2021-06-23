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

func hist(name, help string) prometheus.Histogram {
	return prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      name,
		Help:      help,
		Buckets:   []float64{float64(30 * time.Millisecond), float64(60 * time.Millisecond), float64(120 * time.Millisecond), float64(240 * time.Millisecond), float64(480 * time.Millisecond)},
	})
}

var (
	LeadTime        = gauge("lead_time", "Seconds used in deployment pipeline, from making the request until the application is available")
	TimeSinceDeploy = gauge("time_since_deploy", "Seconds since the latest deploy of this application")

	DeployTimestamp = gauge("deploy_timestamp", "Timestamp when the deploy of this application was triggered in the pipeline")
	StartTimestamp  = gauge("start_timestamp", "Start time of the application")
	BucketWrite     = gauge("bucket_write_latency", "The time it takes to write to the bucket in nanoseconds")
	BucketWriteHist = hist("bucket_write_latency_hist", "The time it takes to write to the bucket in nanoseconds")
	BucketWriteFailed = gauge("bucket_write_requests_failed", "The total of failed bucket writes.")
	BucketRead      = gauge("bucket_read_latency", "The time it takes to read to the bucket in nanoseconds")
	BucketReadHist  = hist("bucket_read_latency_hist", "The time it takes to read to the bucket in nanoseconds")
	BucketReadFailed = gauge("bucket_read_requests_failed", "The total of failed bucket reads.")
	DbInsert        = gauge("db_insert_latency", "The time it takes to insert to table")
	DbRead          = gauge("db_read_latency", "The time it takes to read from table")
)

func init() {
	prometheus.MustRegister(LeadTime)
	prometheus.MustRegister(TimeSinceDeploy)
	prometheus.MustRegister(DeployTimestamp)
	prometheus.MustRegister(StartTimestamp)
	prometheus.MustRegister(BucketWrite)
	prometheus.MustRegister(BucketWriteHist)
	prometheus.MustRegister(BucketWriteFailed)
	prometheus.MustRegister(BucketRead)
	prometheus.MustRegister(BucketReadHist)
	prometheus.MustRegister(BucketReadFailed)
	prometheus.MustRegister(DbInsert)
	prometheus.MustRegister(DbRead)
}

func SetLatencyMetric(start time.Time, gauge prometheus.Gauge) time.Duration {
	latency := time.Since(start)
	gauge.Set(float64(latency.Nanoseconds()))
	return latency
}

func SetLatencyMetricHist(start time.Time, hist prometheus.Histogram) time.Duration {
	latency := time.Since(start)
	hist.Observe(float64(latency.Nanoseconds()))
	return latency
}

func Handler() http.Handler {
	return promhttp.Handler()
}
