package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jhrv/testapp/pkg/version"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	bindAddr string
	gracefulShutdownPeriodSeconds int
)

func init() {
	flag.StringVar(&bindAddr, "bind-address", ":8080", "ip:port where http requests are served")
	flag.IntVar(&gracefulShutdownPeriodSeconds, "graceful-shutdown-wait", 5, "when receiving interrupt signal, it will wait this amount of seconds before shutting down server")
	flag.Parse()
}

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGTERM, syscall.SIGINT)
	hostname, _ := os.Hostname()

	r := mux.NewRouter()

	r.HandleFunc("/isalive", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "yes")
	})

	r.HandleFunc("/version", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s (rev: %s)", version.Version, version.Revision)
	})

	r.HandleFunc("/hostname", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, hostname)
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
