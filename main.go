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

var bindAddr string

func init() {
	flag.StringVar(&bindAddr, "bind-address", ":8080", "ip:port where http requests are served")
	flag.Parse()
}

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	r := mux.NewRouter()

	r.HandleFunc("/isalive", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "yes")
	})

	r.HandleFunc("/version", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s (rev: %s)", version.Version, version.Revision)
	})

	log.Println("running @", bindAddr)
	server := &http.Server{Addr: bindAddr, Handler: r}

	go func() {
		log.Fatal(server.ListenAndServe())
	}()

	<-interrupt

	log.Print("allowing some time for graceful shutdown")
	time.Sleep(20 * time.Second)
	log.Print("shutting down")
	server.Shutdown(context.Background())
}
