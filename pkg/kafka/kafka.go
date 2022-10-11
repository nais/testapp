package kafka

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	"github.com/Shopify/sarama"
	log "github.com/sirupsen/logrus"
)

type Kafka struct {
	name    string
	config  *sarama.Config
	brokers []*sarama.Broker
}

func (kafka *Kafka) Name() string {
	return "kafka"
}

func NewKafkaTest(brokersString, caPath, certPath, keyPath string) (*Kafka, error) {
	keypair, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	caCert, err := os.ReadFile(caPath)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{keypair},
		RootCAs:      caCertPool,
	}

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Net.TLS.Enable = true
	config.Net.TLS.Config = tlsConfig
	config.Version = sarama.V0_10_2_0

	brokers := make([]*sarama.Broker, 0)
	for _, b := range strings.Split(brokersString, ",") {
		brokers = append(brokers, sarama.NewBroker(b))
	}

	return &Kafka{
		config:  config,
		brokers: brokers,
	}, nil
}

func (kafka *Kafka) Init(ctx context.Context) error {
	return nil
}

func (kafka *Kafka) Cleanup() {
	for _, b := range kafka.brokers {
		_ = b.Close()
	}
}

func (kafka *Kafka) Test(ctx context.Context, data string) (string, error) {
	for _, b := range kafka.brokers {
		if err := b.Open(kafka.config); err != nil {
			return "", fmt.Errorf("opening connection to broker: %s: %w", b.Addr(), err)
		}
		connected, err := b.Connected()
		if err != nil || !connected {
			return "", fmt.Errorf("verifying connection to broker: %s: %w", b.Addr(), err)

		}
		if err := b.Close(); err != nil {
			return "", fmt.Errorf("could not close connection: %w", err)
		}
	}
	return data, nil
}
