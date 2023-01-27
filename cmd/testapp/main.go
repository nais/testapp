package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/common/version"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	bindAddr     string
	pingResponse string
	connectURL   string
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})

	flag.StringVar(&bindAddr, "bind-address", ":8080", "ip:port where http requests are served")
	flag.StringVar(&pingResponse, "ping-response", "pong\n", "what to respond when pinged")
	flag.StringVar(&connectURL, "connect-url", "https://google.com", "URL to connect to with /connect")
	flag.Parse()
}

func main() {
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		param := r.URL.Query().Get("delay")
		if d, err := time.ParseDuration(param); err == nil {
			time.Sleep(d)
		}

		_, _ = fmt.Fprint(w, pingResponse)
	})

	http.HandleFunc("/version", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, "%s (rev: %s)", version.Version, version.Revision)
	})

	http.HandleFunc("/hostname", func(w http.ResponseWriter, _ *http.Request) {
		hostname, _ := os.Hostname()
		_, _ = fmt.Fprint(w, hostname)
	})

	http.HandleFunc("/env", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, os.Environ())
	})

	http.HandleFunc("/loginfo", func(w http.ResponseWriter, _ *http.Request) {
		log.Info("info log entry from testapp")
	})

	http.HandleFunc("/logerror", func(w http.ResponseWriter, _ *http.Request) {
		log.Error("error log entry from testapp")
	})

	http.HandleFunc("/logdebug", func(w http.ResponseWriter, _ *http.Request) {
		log.Debug("debug log entry from testapp")
	})

	http.HandleFunc("/connect", func(w http.ResponseWriter, _ *http.Request) {
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

	log.SetLevel(log.DebugLevel)

	log.Infof("running @ %s", bindAddr)

	if err := http.ListenAndServe(bindAddr, nil); err != nil {
		log.Fatal(err)
	}
}
