package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/nais/testapp/pkg/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	bindAddr                      string
	pingResponse                  string
	connectURL                    string
	gracefulShutdownPeriodSeconds int
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})

	flag.StringVar(&bindAddr, "bind-address", ":8080", "ip:port where http requests are served")
	flag.StringVar(&pingResponse, "ping-response", "pong\n", "what to respond when pinged")
	flag.StringVar(&connectURL, "connect-url", "https://google.com", "URL to connect to with /connect")
	flag.IntVar(&gracefulShutdownPeriodSeconds, "graceful-shutdown-wait", 0, "when receiving interrupt signal, it will wait this amount of seconds before shutting down server")
	flag.Parse()
}

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGTERM, syscall.SIGINT)
	hostname, _ := os.Hostname()

	r := mux.NewRouter()

	r.Handle("/metrics", promhttp.Handler())

	r.HandleFunc("/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, pingResponse)
	})

	r.HandleFunc("/version", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s (rev: %s)", version.Version, version.Revision)
	})

	r.HandleFunc("/hostname", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, hostname)
	})

	r.HandleFunc("/env", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, os.Environ())
	})

	r.HandleFunc("/log", func(w http.ResponseWriter, _ *http.Request) {
		log.Info("this is a log statement from testapp")
		w.WriteHeader(http.StatusOK)
	})

	r.HandleFunc("/logerror", func(w http.ResponseWriter, _ *http.Request) {
		log.Error("this is a error log statement from testapp")
		w.WriteHeader(http.StatusOK)
	})

	r.HandleFunc("/connect", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

		resp, err := http.Get(connectURL)
		if err != nil {
			log.Error("error performing http get with url", connectURL, err)
		}

		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Error("error reading response body", err)
		}
		fmt.Fprintf(w, "HTTP status: %d, body:\n%s", resp.StatusCode, string(b))
	})

	log.Println("running @", bindAddr)
	server := &http.Server{Addr: bindAddr, Handler: r}

	go func() {
		log.Fatal(server.ListenAndServe())
	}()

	<-interrupt

	log.Printf("allowing %d seconds to shut down gracefully", gracefulShutdownPeriodSeconds)
	time.Sleep(time.Duration(gracefulShutdownPeriodSeconds) * time.Duration(time.Second))
	log.Print("shutting down")
	server.Shutdown(context.Background())
}
