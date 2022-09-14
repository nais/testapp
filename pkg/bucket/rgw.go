package bucket

import (
	"bytes"
	"context"
	"fmt"
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
	pathStyle  bool   `ceph url style, true means you can use host directly,  
	false means bucketName.doname will be used`
	accessKey string // `aws s3 aceessKey`
	secretKey string // `aws s3 secretKey`
	region    string // `aws s3 region`
	blockSize int64  `blockSize`
}

type Ceph struct {
	client     *s3.S3
	objectName string
	config     *CephConfig
}

func (c *Ceph) Name() string {
	return "ceph"
}

func NewCephBucketTest(host, bucket, region, accessKey, secretKey, objectName string) (*Ceph, error) {
	config := &CephConfig{
		host:       host,
		bucketName: bucket,
		accessKey:  accessKey,
		secretKey:  secretKey,
		region:     region,
		pathStyle:  true,
		blockSize:  8 * 1024 * 1024,
	}

	return &Ceph{
		client:     createServiceClient(config),
		objectName: objectName,
		config:     config,
	}, nil
}

func (c *Ceph) Retrieve() (credentials.Value, error) {
	return credentials.Value{
		AccessKeyID:     c.config.accessKey,
		SecretAccessKey: c.config.secretKey,
	}, nil
}

func (c *Ceph) IsExpired() bool { return false }

func (c *Ceph) Test(ctx context.Context, data string) (string, error) {
	err := c.write(data)
	if err != nil {
		return "", err
	}

	return c.read()
}

func (c *Ceph) Init(ctx context.Context) error {
	return nil
}

func (c *Ceph) Cleanup() {
}

func (c *Ceph) read() (string, error) {
	start := time.Now()

	object, err := c.client.GetObject(
		&s3.GetObjectInput{
			Bucket: aws.String(c.config.bucketName),
			Key:    aws.String(c.objectName),
		})

	if err != nil {
		metrics.BucketReadFailed.Inc()
		return "", fmt.Errorf("unable to read from bucket: %s", err)
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(object.Body)
	if err != nil {
		metrics.BucketReadFailed.Inc()
		return "", fmt.Errorf("unable to read object body: %s", err)
	}

	latency := time.Since(start)
	metrics.RgwReadHist.Observe(float64(latency.Nanoseconds()))
	metrics.RgwRead.Set(float64(latency.Nanoseconds()))
	log.Debugf("read from bucket took %d ms", latency.Milliseconds())

	return buf.String(), nil
}

func (c *Ceph) write(data string) error {
	start := time.Now()

	_, err := c.client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(c.config.bucketName),
		Key:    aws.String(c.objectName),
		Body:   bytes.NewReader([]byte(data)),
	})

	if err != nil {
		metrics.RgwWriteFailed.Inc()
		return fmt.Errorf("could not write to bucket %s: %v", c.config.bucketName, err)
	}

	latency := time.Since(start)
	metrics.RgwWriteHist.Observe(float64(latency.Nanoseconds()))
	metrics.RgwWrite.Set(float64(latency.Nanoseconds()))
	log.Debugf("write to bucket took %d ms", latency.Milliseconds())
	return nil
}

func createServiceClient(c *CephConfig) *s3.S3 {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Credentials:      credentials.NewCredentials(&Ceph{}),
			Endpoint:         &c.host,
			Region:           &c.region,
			DisableSSL:       aws.Bool(true),
			S3ForcePathStyle: &c.pathStyle,
		},
	}))

	return s3.New(sess)
}
