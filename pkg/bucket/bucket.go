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

		start := time.Now()
		reader, err := client.Bucket(bucketName).Object(bucketObjectName).NewReader(context.Background())
		defer closeStorageReader(reader)

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
		latency := metrics.SetLatencyMetricHist(start, metrics.BucketReadHist)
		_ = metrics.SetLatencyMetric(start, metrics.BucketRead)
		log.Debugf("read from bucket took %d ns", latency.Nanoseconds())

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

		ctx := context.Background()
		client, err := storage.NewClient(ctx)
		if err != nil {
			log.Errorf("error creating storage client: %s", err)
		}
		defer client.Close()

		o := client.Bucket(bucketName).Object(bucketObjectName)
		writer := o.NewWriter(context.Background())
		start := time.Now()
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
		latency := metrics.SetLatencyMetricHist(start, metrics.BucketWriteHist)
		_ = metrics.SetLatencyMetric(start, metrics.BucketWrite)
		log.Debugf("write to bucket took %d ns", latency.Nanoseconds())

		objectAttrsToUpdate := cacheControl("no-store")

		if _, err := o.Update(ctx, objectAttrsToUpdate); err != nil {
			log.Errorf("ObjectHandle(%q).Update: %v", o, err)
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func cacheControl(cacheControl string) storage.ObjectAttrsToUpdate {
	objectAttrsToUpdate := storage.ObjectAttrsToUpdate{
		CacheControl: cacheControl,
	}
	return objectAttrsToUpdate
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
