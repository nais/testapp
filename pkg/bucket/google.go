package bucket

import (
	"context"
	"fmt"
	"google.golang.org/api/iterator"
	"io"
	"os"
	"time"

	"cloud.google.com/go/storage"
	log "github.com/sirupsen/logrus"

	"github.com/nais/testapp/pkg/metrics"
)

type Bucket struct {
	name   string
	client *storage.Client
	object *storage.ObjectHandle
}

func (bucket *Bucket) Name() string {
	return "bucket"
}

func NewGoogleBucketTest(ctx context.Context, bucketName, bucketObjectName string) (*Bucket, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		metrics.BucketWriteFailed.Inc()
		return nil, err
	}

	object := client.Bucket(bucketName).Object(bucketObjectName)

	return &Bucket{
		name:   bucketName,
		client: client,
		object: object,
	}, nil
}

func (bucket *Bucket) Init(ctx context.Context) error {
	it := bucket.client.Buckets(ctx, os.Getenv("GCP_TEAM_PROJECT_ID"))
	for {
		_, err := it.Next()
		if err == iterator.Done {
			return nil
		}
		if err != nil {
			return fmt.Errorf("unable to list buckets: %s", err)
		}
	}
}

func (bucket *Bucket) Cleanup() {
	err := bucket.client.Close()
	if err != nil {
		log.Errorf("cleanup bucket: %v", err)
	}
}

func (bucket *Bucket) Test(ctx context.Context, data string) (string, error) {
	err := bucket.write(ctx, data)
	if err != nil {
		return "", err
	}

	return bucket.read(ctx)
}

func (bucket *Bucket) read(ctx context.Context) (string, error) {
	start := time.Now()
	reader, err := bucket.object.NewReader(ctx)
	defer closeStorageReader(reader)

	if err != nil {
		metrics.BucketReadFailed.Inc()
		return "", fmt.Errorf("unable to create reader: %s", err)
	}

	res, err := io.ReadAll(reader)
	if err != nil {
		metrics.BucketReadFailed.Inc()
		return "", fmt.Errorf("unable to read from bucket: %s", err)
	}

	latency := time.Since(start)
	metrics.BucketReadHist.Observe(float64(latency.Nanoseconds()))
	metrics.BucketRead.Set(float64(latency.Nanoseconds()))
	log.Debugf("read from bucket took %d ms", latency.Milliseconds())

	return string(res), nil
}

func (bucket *Bucket) write(ctx context.Context, content string) error {
	writer := bucket.object.NewWriter(ctx)
	start := time.Now()
	defer closeStorageWriter(writer)

	_, err := writer.Write([]byte(content))
	if err != nil {
		metrics.BucketWriteFailed.Inc()
		return fmt.Errorf("unable to write to bucket: %s", err)
	}

	latency := time.Since(start)
	metrics.BucketWriteHist.Observe(float64(latency.Nanoseconds()))
	metrics.BucketWrite.Set(float64(latency.Nanoseconds()))
	log.Debugf("write to bucket took %d ms", latency.Milliseconds())

	objectAttrsToUpdate := cacheControl("no-store")

	if _, err := bucket.object.Update(ctx, objectAttrsToUpdate); err != nil {
		return fmt.Errorf("ObjectHandle(%q).Update: %v", bucket.object.ObjectName(), err)
	}

	return nil
}

func cacheControl(cacheControl string) storage.ObjectAttrsToUpdate {
	objectAttrsToUpdate := storage.ObjectAttrsToUpdate{
		CacheControl: cacheControl,
	}
	return objectAttrsToUpdate
}

func closeStorageWriter(reader *storage.Writer) {
	if reader != nil {
		err := reader.Close()

		if err != nil {
			log.Errorf("Failed to close storage writer: %s", err)
		}
	} else {
		log.Warn("Attempted to close nil reader")
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
