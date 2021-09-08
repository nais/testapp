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

func hist(name, help string, buckets []float64) prometheus.Histogram {
	return prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      name,
		Help:      help,
		Buckets:   buckets,
	})
}

var (
	LeadTime        = gauge("lead_time", "Seconds used in deployment pipeline, from making the request until the application is available")
	TimeSinceDeploy = gauge("time_since_deploy", "Seconds since the latest deploy of this application")

	DeployTimestamp  = gauge("deploy_timestamp", "Timestamp when the deploy of this application was triggered in the pipeline")
	StartTimestamp   = gauge("start_timestamp", "Start time of the application")
	BucketWrite      = gauge("bucket_write_latency", "The time it takes to write to the bucket in nanoseconds")
	writeHistBuckets = []float64{float64(10 * time.Millisecond), float64(20 * time.Millisecond),
		float64(30 * time.Millisecond), float64(40 * time.Millisecond), float64(50 * time.Millisecond),
		float64(60 * time.Millisecond), float64(70 * time.Millisecond), float64(80 * time.Millisecond),
		float64(90 * time.Millisecond), float64(100 * time.Millisecond)}
	BucketWriteHist   = hist("bucket_write_latency_hist", "The time it takes to write to the bucket in nanoseconds", writeHistBuckets)
	BucketWriteFailed = gauge("bucket_write_requests_failed", "The total of failed bucket writes.")
	BucketRead        = gauge("bucket_read_latency", "The time it takes to read to the bucket in nanoseconds")
	readHistBuckets   = []float64{float64(10 * time.Millisecond), float64(20 * time.Millisecond),
		float64(30 * time.Millisecond), float64(40 * time.Millisecond), float64(50 * time.Millisecond),
		float64(60 * time.Millisecond), float64(70 * time.Millisecond), float64(80 * time.Millisecond),
		float64(90 * time.Millisecond), float64(100 * time.Millisecond)}
	BucketReadHist   = hist("bucket_read_latency_hist", "The time it takes to read to the bucket in nanoseconds", readHistBuckets)
	BucketReadFailed = gauge("bucket_read_requests_failed", "The total of failed bucket reads.")
	RgwWrite         = gauge("rgw_write_latency", "The time it takes to write to the ceph bucket in nanoseconds")
	writeHistRgw     = []float64{float64(10 * time.Millisecond), float64(20 * time.Millisecond),
		float64(30 * time.Millisecond), float64(40 * time.Millisecond), float64(50 * time.Millisecond),
		float64(60 * time.Millisecond), float64(70 * time.Millisecond), float64(80 * time.Millisecond),
		float64(90 * time.Millisecond), float64(100 * time.Millisecond)}
	RgwWriteHist = hist("rgw_write_latency_hist", "The time it takes to write to the ceph bucket in nanoseconds",
		writeHistRgw)
	RgwWriteFailed = gauge("rgw_write_requests_failed", "The total of failed ceph bucket writes.")
	RgwRead        = gauge("rgw_read_latency", "The time it takes to read to the ceph bucket in nanoseconds")
	readHistRgw    = []float64{float64(10 * time.Millisecond), float64(20 * time.Millisecond),
		float64(30 * time.Millisecond), float64(40 * time.Millisecond), float64(50 * time.Millisecond),
		float64(60 * time.Millisecond), float64(70 * time.Millisecond), float64(80 * time.Millisecond),
		float64(90 * time.Millisecond), float64(100 * time.Millisecond)}
	RgwReadHist = hist("rgw_read_latency_hist", "The time it takes to read to the ceph bucket in nanoseconds",
		readHistRgw)
	RgwReadFailed       = gauge("rgw_read_requests_failed", "The total of failed ceph bucket reads.")
	DbInsert            = gauge("db_insert_latency", "The time it takes to insert to table")
	dbInsertHistBuckets = []float64{float64(5 * time.Millisecond), float64(10 * time.Millisecond),
		float64(15 * time.Millisecond), float64(20 * time.Millisecond), float64(25 * time.Millisecond),
		float64(30 * time.Millisecond), float64(35 * time.Millisecond), float64(40 * time.Millisecond),
		float64(45 * time.Millisecond), float64(50 * time.Millisecond)}
	DbInsertHist      = hist("db_insert_latency_hist", "The time it takes to insert to table", dbInsertHistBuckets)
	DbInsertFailed    = gauge("db_insert_failed", "The total of failed db inserts.")
	DbRead            = gauge("db_read_latency", "The time it takes to read from table")
	dbReadHistBuckets = []float64{float64(5 * time.Millisecond), float64(10 * time.Millisecond),
		float64(15 * time.Millisecond), float64(20 * time.Millisecond), float64(25 * time.Millisecond),
		float64(30 * time.Millisecond), float64(35 * time.Millisecond), float64(40 * time.Millisecond),
		float64(45 * time.Millisecond), float64(50 * time.Millisecond)}
	DbReadHist   = hist("db_read_latency_hist", "The time it takes to read from table", dbReadHistBuckets)
	DbReadFailed = gauge("db_read_failed", "The total of failed db reads.")

	BqInsert            = gauge("bq_insert_latency", "The time it takes to insert to table")
	bqInsertHistBuckets = []float64{float64(2000 * time.Millisecond), float64(4000 * time.Millisecond), float64(8000 * time.Millisecond)}
	BqInsertHist        = hist("bq_insert_latency_hist", "The time it takes to insert to table", bqInsertHistBuckets)
	BqInsertFailed      = gauge("bq_insert_failed", "The total of failed db inserts.")
	BqRead              = gauge("bq_read_latency", "The time it takes to read from table")
	bqReadHistBuckets   = []float64{float64(500 * time.Millisecond), float64(1000 * time.Millisecond), float64(2000 * time.Millisecond)}
	BqReadHist          = hist("bq_read_latency_hist", "The time it takes to read from table", bqReadHistBuckets)
	BqReadFailed        = gauge("dbq_read_failed", "The total of failed db reads.")
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
	prometheus.MustRegister(RgwWrite)
	prometheus.MustRegister(RgwWriteHist)
	prometheus.MustRegister(RgwWriteFailed)
	prometheus.MustRegister(RgwRead)
	prometheus.MustRegister(RgwReadHist)
	prometheus.MustRegister(RgwReadFailed)
	prometheus.MustRegister(DbInsert)
	prometheus.MustRegister(DbInsertHist)
	prometheus.MustRegister(DbInsertFailed)
	prometheus.MustRegister(DbRead)
	prometheus.MustRegister(DbReadHist)
	prometheus.MustRegister(DbReadFailed)
	prometheus.MustRegister(BqInsert)
	prometheus.MustRegister(BqInsertHist)
	prometheus.MustRegister(BqInsertFailed)
	prometheus.MustRegister(BqRead)
	prometheus.MustRegister(BqReadHist)
	prometheus.MustRegister(BqReadFailed)
}

func Handler() http.Handler {
	return promhttp.Handler()
}
