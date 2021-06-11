package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/nais/testapp/pkg/bigquery"
	"github.com/nais/testapp/pkg/bucket"
	"github.com/nais/testapp/pkg/database"
	"github.com/nais/testapp/pkg/metrics"
	"github.com/nais/testapp/pkg/version"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	bindAddr                      string
	pingResponse                  string
	connectURL                    string
	gracefulShutdownPeriodSeconds int
	deployStartTimestamp          int64
	bucketName                    string
	bucketObjectName              string
	dbUser                        string
	dbPassword                    string
	dbHost                        string
	dbName                        string
	bigqueryName                  string
	projectID                     string
	debug                         bool
)

var (
	dbAppName         = strings.ToUpper(strings.Replace(getEnv("NAIS_APP_NAME", "TESTAPP"), "-", "_", -1))
	defaultDbPassword = os.Getenv(fmt.Sprintf("NAIS_DATABASE_%[1]s_%[1]s_PASSWORD", dbAppName))
	defaultDbUsername = os.Getenv(fmt.Sprintf("NAIS_DATABASE_%[1]s_%[1]s_USERNAME", dbAppName))
	defaultDbName     = os.Getenv(fmt.Sprintf("NAIS_DATABASE_%[1]s_%[1]s_DATABASE", dbAppName))
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})

	flag.StringVar(&bindAddr, "bind-address", ":8080", "ip:port where http requests are served")
	flag.StringVar(&pingResponse, "ping-response", "pong\n", "what to respond when pinged")
	flag.StringVar(&bucketName, "bucket-name", os.Getenv("BUCKET_NAME"), "name of bucket used with /{read,write}bucket")
	flag.StringVar(&projectID, "projectid", os.Getenv("GCP_TEAM_PROJECT_ID"), "projectid used with /{read,write}bigquery")
	flag.StringVar(&bigqueryName, "bigqueryname", os.Getenv("BIGQUERY_NAME"), "name of bigquery dataset used with /{read,write}bigquery")
	flag.StringVar(&bucketObjectName, "bucket-object-name", "test", "name of bucket object used with /{read,write}bucket")
	flag.StringVar(&connectURL, "connect-url", "https://google.com", "URL to connect to with /connect")
	flag.StringVar(&dbName, "db-name", defaultDbName, "database name")
	flag.StringVar(&dbUser, "db-user", defaultDbUsername, "database username")
	flag.StringVar(&dbPassword, "db-password", defaultDbPassword, "database password")
	flag.StringVar(&dbHost, "db-hostname", "localhost", "database hostname")
	flag.BoolVar(&debug, "debug", getEnvBool("DEBUG", false), "debug log")
	flag.IntVar(&gracefulShutdownPeriodSeconds, "graceful-shutdown-wait", 0, "when receiving interrupt signal, it will wait this amount of seconds before shutting down server")
	flag.Int64Var(&deployStartTimestamp, "deploy-start-time", getEnvInt("DEPLOY_START", time.Now().UnixNano()), "unix timestamp with nanoseconds, specifies when NAIS deploy of testapp started")
	flag.Parse()
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		b, _ := strconv.ParseBool(value)
		return b
	}
	return fallback
}

func getEnvInt(key string, fallback int64) int64 {
	if value, ok := os.LookupEnv(key); ok {
		i, _ := strconv.ParseInt(value, 10, 64)
		return i
	}

	return fallback
}

func timeSinceDeploy() float64 {
	deployStartTime := time.Unix(0, deployStartTimestamp)
	return time.Now().Sub(deployStartTime).Seconds()
}

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGTERM, syscall.SIGINT)
	hostname, _ := os.Hostname()

	metrics.StartTimestamp.SetToCurrentTime()
	metrics.DeployTimestamp.Set(float64(deployStartTimestamp) / 10e8)

	metrics.LeadTime.Set(timeSinceDeploy())
	metrics.TimeSinceDeploy.Set(timeSinceDeploy())
	tick := time.NewTicker(time.Second)
	go func() {
		for range tick.C {
			metrics.TimeSinceDeploy.Set(timeSinceDeploy())
		}
	}()

	r := mux.NewRouter()

	r.Handle("/metrics", metrics.Handler())

	r.HandleFunc("/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, pingResponse)
	})

	r.HandleFunc("/version", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "%s (rev: %s)", version.Version, version.Revision)
	})

	r.HandleFunc("/hostname", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, hostname)
	})

	r.HandleFunc("/env", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, os.Environ())
	})

	r.HandleFunc("/log", func(w http.ResponseWriter, _ *http.Request) {
		log.Info("this is a log statement from testapp")
		w.WriteHeader(http.StatusOK)
	})

	r.HandleFunc("/logerror", func(w http.ResponseWriter, _ *http.Request) {
		log.Error("this is a error log statement from testapp")
		w.WriteHeader(http.StatusOK)
	})

	r.HandleFunc("/logdebug", func(w http.ResponseWriter, _ *http.Request) {
		if debug {
			log.Debug("this is a debug log statement from testapp")
		} else {
			log.Info("this would have been a debug log statement from testapp if debug was enabled")
		}
		w.WriteHeader(http.StatusOK)
	})

	r.HandleFunc("/header-test", func(w http.ResponseWriter, r *http.Request) {
		log.Infof("Headers: %+v", r.Header)
		w.Header().Add("X-Frame-Options", "SAMEORIGIN")
		w.Header().Add("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Add("X-Content-Type-Options", "nosniff")
		w.Header().Add("X-XSS-Protection", "1; mode=block")
		w.Header().Add("Referrer-Policy", "no-referrer-when-downgrade")

		w.WriteHeader(http.StatusOK)
	})

	r.HandleFunc("/connect", func(w http.ResponseWriter, _ *http.Request) {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

		resp, err := http.Get(connectURL)
		if err != nil {
			log.Error("error performing http get with url", connectURL, err)
			_, _ = fmt.Fprintf(w, "error performing http get")
			return
		}

		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Error("error reading response body", err)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprint(w, "error reading response body")
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "HTTP status: %d, body:\n%s", resp.StatusCode, string(b))
	})
	r.HandleFunc("/readbucket", bucket.ReadBucketHandler(bucketName, bucketObjectName))
	r.HandleFunc("/writebucket", bucket.WriteBucketHandler(bucketName, bucketObjectName)).Methods(http.MethodPost)
	r.HandleFunc("/writedb", database.WriteDatabaseHandler(dbUser, dbPassword, dbName, dbHost)).Methods(http.MethodPost)
	r.HandleFunc("/readdb", database.ReadDatabaseHandler(dbUser, dbPassword, dbName, dbHost))
	r.HandleFunc("/readbigquery", bigquery.ReadBigQueryHandler(projectID, bigqueryName))
	r.HandleFunc("/writebigquery", bigquery.WriteBigQueryHandler(projectID, bigqueryName)).Methods(http.MethodPost)

	if debug {
		log.SetLevel(log.DebugLevel)
	}
	log.Info("running @", bindAddr)
	server := &http.Server{Addr: bindAddr, Handler: r}

	go func() {
		log.Fatal(server.ListenAndServe())
	}()

	<-interrupt

	log.Printf("allowing %d seconds to shut down gracefully", gracefulShutdownPeriodSeconds)
	time.Sleep(time.Duration(gracefulShutdownPeriodSeconds) * time.Second)
	log.Print("shutting down")

	_ = server.Shutdown(context.Background())
}
