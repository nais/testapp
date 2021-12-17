package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"

	"github.com/nais/testapp/pkg/bigquery"
	"github.com/nais/testapp/pkg/bucket"
	"github.com/nais/testapp/pkg/database"
	"github.com/nais/testapp/pkg/metrics"
	"github.com/nais/testapp/pkg/testable"
	"github.com/nais/testapp/pkg/version"
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
	bigqueryTableName             string
	projectID                     string
	debug                         bool
	rgwAddress                    string
	rgwAccessKey                  string
	rgwSecretKey                  string
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
	flag.StringVar(&bigqueryTableName, "bigquerytablename", os.Getenv("BIGQUERY_TABLE_NAME"), "name of bigquery dataset's table used with /{read,write}bigquery")
	flag.StringVar(&bucketObjectName, "bucket-object-name", "test", "name of bucket object used with /{read,write}bucket")
	flag.StringVar(&rgwAddress, "rgw-address", os.Getenv("RGW_ADDRESS"), "Ceph RGW objectstore address")
	flag.StringVar(&rgwAccessKey, "rgw-access-key", os.Getenv("RGW_ACCESS_KEY"), "Ceph RGW objectstore access key")
	flag.StringVar(&rgwSecretKey, "rgw-secret-key", os.Getenv("RGW_SECRET_KEY"), "Ceph RGW objectstore secret key")
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
	programContext, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	r.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		param := r.URL.Query().Get("delay")
		if d, err := time.ParseDuration(param); err == nil {
			time.Sleep(d)
		}

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

	r.HandleFunc("/logevent", func(w http.ResponseWriter, r *http.Request) {
		fields := make(map[string]interface{})
		for key, value := range r.URL.Query() {
			fields[key] = value[0]
		}
		log.WithField("logtype", "event").WithFields(fields).Info("this is a event log statement")
		w.WriteHeader(http.StatusOK)
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

	// Holds all tests. Possible feature: /test to run all tests
	var tests []testable.Testable

	// Set up google bucket test
	bucketTest, err := bucket.NewGoogleBucketTest(programContext, bucketName, bucketObjectName)
	if err != nil {
		log.Errorf("Error setting up google bucket test: %v", err)
	} else {
		tests = append(tests, bucketTest)
	}

	// Set up ceph test
	if rgwAddress != "" && rgwAccessKey != "" && rgwSecretKey != "" {
		cephTest, err := bucket.NewCephBucketTest(rgwAddress, bucketName, "us-east-1", rgwAccessKey, rgwSecretKey, bucketObjectName)
		if err != nil {
			log.Errorf("Error setting up ceph bucket test: %v", err)
		} else {
			tests = append(tests, cephTest)
		}
	}

	// Set up database test
	databaseTest, err := database.NewDatabaseTest(dbUser, dbPassword, dbName, dbHost)
	if err != nil {
		log.Errorf("Error setting up database test: %v", err)
	} else {
		tests = append(tests, databaseTest)
	}

	// Set up bigquery test
	if bigqueryName != "" && bigqueryTableName != "" {
		bq, err := bigquery.NewBigqueryTest(programContext, projectID, bigqueryName, bigqueryTableName)
		err = bq.Init(programContext)
		if err != nil {
			log.Errorf("Error setting up bigquery test: %v", err)
		} else {
			tests = append(tests, bq)
		}
	}

	for _, test := range tests {
		err := test.Init(programContext)
		if err != nil {
			log.Errorf("Error initializing test: %s, will not set up handler. err: %v", test.Name(), err)
		} else {
			setupTestHandler(r, test)

			//goland:noinspection GoDeferInLoop
			defer test.Cleanup()
		}
	}

	if debug {
		log.SetLevel(log.DebugLevel)
	}
	log.Info("running @", bindAddr)
	server := &http.Server{Addr: bindAddr, Handler: r}

	go func() {
		err := server.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Errorf("closing http server: %v", err)
		}
	}()

	<-interrupt

	log.Printf("allowing %d seconds to shut down gracefully", gracefulShutdownPeriodSeconds)
	time.Sleep(time.Duration(gracefulShutdownPeriodSeconds) * time.Second)
	log.Print("shutting down")

	_ = server.Shutdown(programContext)
}

func setupTestHandler(router *mux.Router, test testable.Testable) {
	log.Infof("setting up %s test handler", test.Name())

	path := fmt.Sprintf("/%s/test", test.Name())
	router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		expected := fmt.Sprintf("%x", 9999+rand.Intn(999999))[:4]
		result, err := test.Test(r.Context(), expected)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, respondErr := fmt.Fprintf(w, "%s test: error: %v", test.Name(), err)
			if respondErr != nil {
				log.Errorf("%s test: write response err: %v (test err: %v)", test.Name(), respondErr, err)
			} else {
				log.Warnf("%s test: failed, %v", test.Name(), err)
			}
			return
		}

		if expected != result {
			w.WriteHeader(http.StatusInternalServerError)
			_, err := fmt.Fprintf(w, "%s test: data missmatch, epected: %s got: %s", test.Name(), expected, result)
			if err != nil {
				log.Errorf("%s test: unable to write response: %v", test.Name(), err)
			}
		} else {
			w.WriteHeader(http.StatusOK)
		}
	})
}
