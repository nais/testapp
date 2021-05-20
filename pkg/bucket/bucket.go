package bucket

import (
	"context"
	"io/ioutil"
	"net/http"
	"time"

	"cloud.google.com/go/storage"
	"github.com/nais/testapp/pkg/metrics"
	log "github.com/sirupsen/logrus"
)

func ReadBucketHandler(bucketName, bucketObjectName string) func(w http.ResponseWriter, _ *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		client, err := storage.NewClient(context.Background())
		if err != nil {
			log.Errorf("error creating storage client: %s", err)
		}

		reader, err := client.Bucket(bucketName).Object(bucketObjectName).NewReader(context.Background())
		defer closeStorageReader(reader)

		if err != nil {
			log.Errorf("unable to create reader: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		start := time.Now()
		res, err := ioutil.ReadAll(reader)
		if err != nil {
			log.Errorf("unable to read from bucket: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		latency := time.Since(start)
		latencyInMs := float64(latency.Nanoseconds()/1000000)
		log.Infof("read from bucket took %d ms", latencyInMs)
		metrics.BucketRead.Observe(latencyInMs)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(res)
	}

}

func WriteBucketHandler(bucketName, bucketObjectName string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		d := string(body)
		if len(d) > 5 || len(d) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("cannot be empty or more than 5 characters"))
			return
		}
		defer r.Body.Close()

		client, err := storage.NewClient(context.Background())
		if err != nil {
			log.Errorf("error creating storage client: %s", err)
		}

		writer := client.Bucket(bucketName).Object(bucketObjectName).NewWriter(context.Background())
		start := time.Now()
		_, err = writer.Write([]byte(d))
		if err != nil {
			log.Errorf("unable to write to bucket: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		latency := time.Since(start)
		latencyInMs := float64(latency.Nanoseconds()/1000000)
		log.Infof("write to bucket took %d ms", latencyInMs)
		metrics.BucketWrite.Observe(latencyInMs)

		if err := writer.Close(); err != nil {
			log.Errorf("unable to close bucket writer: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}
}

func closeStorageReader(reader *storage.Reader) {
	if reader != nil {
		err := reader.Close()

		if err != nil {
			log.Errorf("Failed to close storage reader: %s", err)
		}
	} else {
		log.Warn("Attempted to close nil reader")
	}
}
