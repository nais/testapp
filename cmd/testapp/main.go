package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/nais/testapp/internal/metrics"
	"github.com/prometheus/common/version"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	bindAddr             string
	pingResponse         string
	connectURL           string
	deployStartTimestamp int64
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})

	flag.StringVar(&bindAddr, "bind-address", ":8080", "ip:port where http requests are served")
	flag.StringVar(&pingResponse, "ping-response", "pong\n", "what to respond when pinged")
	flag.StringVar(&connectURL, "connect-url", "https://google.com", "URL to connect to with /connect")
	flag.Int64Var(&deployStartTimestamp, "deploy-start-time", getEnvInt("DEPLOY_START", time.Now().UnixNano()), "unix timestamp with nanoseconds, specifies when NAIS deploy of testapp started")
	flag.Parse()
}

func setupHandlers(mux *http.ServeMux) {
	mux.Handle("/metrics", metrics.Handler())

	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		param := r.URL.Query().Get("delay")
		if d, err := time.ParseDuration(param); err == nil {
			time.Sleep(d)
		}

		_, _ = fmt.Fprint(w, pingResponse)
	})

	mux.HandleFunc("/version", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, "%s (rev: %s)", version.Version, version.Revision)
	})

	mux.HandleFunc("/hostname", func(w http.ResponseWriter, _ *http.Request) {
		hostname, _ := os.Hostname()
		_, _ = fmt.Fprint(w, hostname)
	})

	mux.HandleFunc("/env", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, os.Environ())
	})

	mux.HandleFunc("/loginfo", func(w http.ResponseWriter, _ *http.Request) {
		log.Info("info log entry from testapp")
	})

	mux.HandleFunc("/logerror", func(w http.ResponseWriter, _ *http.Request) {
		log.Error("error log entry from testapp")
	})

	mux.HandleFunc("/logdebug", func(w http.ResponseWriter, _ *http.Request) {
		log.Debug("debug log entry from testapp")
	})

	mux.HandleFunc("/connect", func(w http.ResponseWriter, _ *http.Request) {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

		resp, err := http.Get(connectURL)
		if err != nil {
			log.Error("error performing http get with url", connectURL, err)
			_, _ = fmt.Fprintf(w, "error performing http get")
			return
		}

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Error("error reading response body", err)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprint(w, "error reading response body")
			return
		}

		_, _ = fmt.Fprintf(w, "HTTP status: %d, body:\n%s", resp.StatusCode, string(b))
	})
}

func main() {
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

	setupHandlers(http.DefaultServeMux)

	log.SetLevel(log.DebugLevel)

	log.Infof("running @ %s", bindAddr)

	if err := http.ListenAndServe(bindAddr, nil); err != nil {
		log.Fatal(err)
	}
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
	return time.Since(deployStartTime).Seconds()
}
