package bucket

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/nais/testapp/pkg/metrics"

	log "github.com/sirupsen/logrus"
)

type CephConfig struct {
	host       string `ceph host`
	bucketName string `bucketName`
	pathStyle   bool  `ceph url style, true means you can use host directly,  
	false means bucketName.doname will be used`
	accessKey string // `aws s3 aceessKey`
	secretKey string // `aws s3 secretKey`
	region    string // `aws s3 region`
	blockSize int64  `blockSize`
}

var Ceph CephConfig

type CephProvider struct{}

func (c *CephProvider) Retrieve() (credentials.Value, error) {
	return credentials.Value{
		AccessKeyID:     Ceph.accessKey,
		SecretAccessKey: Ceph.secretKey,
	}, nil
}

func (c *CephProvider) IsExpired() bool { return false }

func (c *CephConfig) CephInit(host, bucket, region, accessKey, secretKey string) error {
	c.host = host
	c.bucketName = bucket
	c.accessKey = accessKey
	c.secretKey = secretKey
	c.region = region
	c.pathStyle = true
	c.blockSize = 8 * 1024 * 1024

	return nil
}

func (c *CephConfig) ReadBucketHandler(key string) func(w http.ResponseWriter, _ *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		s3Service, err := c.createServiceClient()
		if err != nil {
			log.Errorf("error creating storage client: %v", err)
			metrics.BucketReadFailed.Inc()
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		start := time.Now()
		object, err := s3Service.GetObject(
			&s3.GetObjectInput{
				Bucket: aws.String(c.bucketName),
				Key:    aws.String(key),
			})

		if err != nil {
			log.Errorf("unable to read from bucket: %s", err)
			metrics.BucketReadFailed.Inc()
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(object.Body)
		if err != nil {
			log.Errorf("unable to read object body: %s", err)
			metrics.BucketReadFailed.Inc()
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		latency := float64(time.Since(start).Nanoseconds())
		metrics.RgwReadHist.Observe(latency)
		metrics.RgwRead.Set(latency)
		log.Debugf("read from bucket took %d ns", latency)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}
}

func (c *CephConfig) WriteBucketHandler(key string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if len(string(body)) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("cannot be empty"))
			return
		}
		defer r.Body.Close()

		s3Service, err := c.createServiceClient()
		if err != nil {
			log.Errorf("error creating storage client: %v", err)
			metrics.RgwWriteFailed.Inc()
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		start := time.Now()

		_, err = s3Service.PutObject(&s3.PutObjectInput{
				Bucket: aws.String(c.bucketName),
				Key:    aws.String(key),
				Body:   bytes.NewReader(body),
			})

		if err != nil {
			log.Errorf("could not write to bucket %s: %v", c.bucketName, err)
			metrics.RgwWriteFailed.Inc()
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		latency := float64(time.Since(start).Nanoseconds())
		metrics.RgwWriteHist.Observe(latency)
		metrics.RgwWrite.Set(latency)
		log.Debugf("write to bucket took %d ns", latency)

		w.WriteHeader(http.StatusOK)
	}
}

func (c *CephConfig) createServiceClient() (*s3.S3, error) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Credentials:                       credentials.NewCredentials(&CephProvider{}),
			Endpoint:                          &c.host,
			Region:                            &c.region,
			DisableSSL:                        aws.Bool(true),
			S3ForcePathStyle:                  &c.pathStyle,
		}}))

	return s3.New(sess), nil
}
