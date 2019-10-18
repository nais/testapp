package main

import (
	"cloud.google.com/go/storage"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/nais/testapp/pkg/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"google.golang.org/api/option"
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
	bucketName                    string
	serviceAccountCredentialsFile string
	bucketObjectName              string
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})

	flag.StringVar(&bindAddr, "bind-address", ":8080", "ip:port where http requests are served")
	flag.StringVar(&pingResponse, "ping-response", "pong\n", "what to respond when pinged")
	flag.StringVar(&bucketName, "bucket-name", os.Getenv("BUCKET_NAME"), "name of bucket used with /{read,write}bucket")
	flag.StringVar(&bucketObjectName, "bucket-object-name", "test", "name of bucket object used with /{read,write}bucket")
	flag.StringVar(&serviceAccountCredentialsFile, "service-account-credentials-file", "/var/run/secrets/testapp-serviceaccount.json", "path to service account credentials file")
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
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

		resp, err := http.Get(connectURL)
		if err != nil {
			log.Error("error performing http get with url", connectURL, err)
			fmt.Fprintf(w, "error performing http get")
			return
		}

		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Error("error reading response body", err)
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "error reading response body")
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "HTTP status: %d, body:\n%s", resp.StatusCode, string(b))
	})

	r.HandleFunc("/readbucket", func(w http.ResponseWriter, _ *http.Request) {
		if err := verifyBucketPrerequisites(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}

		client, err := storage.NewClient(context.Background(), option.WithCredentialsFile(serviceAccountCredentialsFile))
		if err != nil {
			log.Errorf("error creating storage client: %s", err)
		}

		reader, err := client.Bucket(bucketName).Object(bucketObjectName).NewReader(context.Background())
		defer reader.Close()

		if err != nil {
			log.Errorf("unable to create reader: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := ioutil.ReadAll(reader)
		if err != nil {
			log.Errorf("unable to read from bucket: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(res)
	})

	r.HandleFunc("/writebucket", func(w http.ResponseWriter, r *http.Request) {
		if err := verifyBucketPrerequisites(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		d := string(body)
		if len(d) > 5 || len(d) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("cannot be empty or more than 5 characters"))
			return
		}

		client, err := storage.NewClient(context.Background(), option.WithCredentialsFile(serviceAccountCredentialsFile))

		if err != nil {
			log.Errorf("error creating storage client: %s", err)
		}

		writer := client.Bucket(bucketName).Object(bucketObjectName).NewWriter(context.Background())
		_, err = writer.Write([]byte(d))

		if err != nil {
			log.Errorf("unable to write to bucket: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err := writer.Close(); err != nil {
			log.Errorf("unable to close bucket writer: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}).Methods(http.MethodPost)

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

func verifyBucketPrerequisites() error {
	if len(bucketName) == 0 {
		return fmt.Errorf("missing bucket-name")
	}

	if _, err := os.Stat(serviceAccountCredentialsFile); err != nil {
		return fmt.Errorf("missing service account credentials")
	}

	return nil
}
