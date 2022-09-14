package bigquery

import (
	"context"
	"fmt"
	"github.com/nais/testapp/pkg/retry"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"

	"github.com/nais/testapp/pkg/metrics"
)

// Item represents a row item.
type Item struct {
	Message string
}

type BigQuery struct {
	client             *bigquery.Client
	table              *bigquery.Table
	retryContextConfig *retry.ContextConfig
}

func (bq *BigQuery) Name() string {
	return "bigquery"
}

func NewBigqueryTest(ctx context.Context, projectID, datasetID, tableID string, maxRetry, retryInterval int) (*BigQuery, error) {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	retryConfig := retry.NewContextConfig(
		ctx,
		maxRetry,
		retryInterval,
	)

	return &BigQuery{
		client:             client,
		table:              client.Dataset(datasetID).Table(tableID),
		retryContextConfig: retryConfig,
	}, nil
}

func (bq *BigQuery) Test(ctx context.Context, data string) (string, error) {
	defer bq.truncate(ctx)

	err := bq.write(ctx, data)
	if err != nil {
		return "", err
	}

	return bq.read(ctx)
}

func (bq *BigQuery) Cleanup() {
	err := bq.client.Close()
	if err != nil {
		log.Errorf("cleanup bigquery: %v", err)
	}
}

func (bq *BigQuery) read(ctx context.Context) (string, error) {
	start := time.Now()
	tableRows := bq.table.Read(ctx)
	var row Item
	c := 0
	for {
		err := tableRows.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			metrics.BqReadFailed.Inc()
			return "", err
		}
		c += 1
	}

	latency := time.Since(start)
	metrics.BqRead.Set(float64(latency.Nanoseconds()))
	metrics.BqReadHist.Observe(float64(latency.Nanoseconds()))
	log.Debugf("read from big query took %d ms", latency.Milliseconds())

	if c != 1 {
		metrics.BqReadFailed.Inc()
		return "", fmt.Errorf("test returned incorrect amount of data %v", c)
	}

	return row.Message, nil
}

func (bq *BigQuery) write(ctx context.Context, content string) error {
	item := Item{
		Message: content,
	}

	q := bq.client.Query("INSERT INTO `" + strings.ReplaceAll(bq.table.FullyQualifiedName(), ":", ".") + "` VALUES (\"" + item.Message + `")`)
	q.Priority = bigquery.InteractivePriority

	start := time.Now()
	job, err := q.Run(ctx)
	if err != nil {
		metrics.BqInsert.Inc()
		return err
	}

	s, err := job.Wait(ctx)
	if err != nil || s.Err() != nil {
		if err == nil {
			err = s.Err()
		}
		metrics.BqInsertFailed.Inc()
		return err
	}

	latency := time.Since(start)
	metrics.BqInsert.Set(float64(latency.Nanoseconds()))
	metrics.BqInsertHist.Observe(float64(latency.Nanoseconds()))
	log.Debugf("write to big query took %d ms", latency.Milliseconds())
	return nil
}

func (bq *BigQuery) truncate(ctx context.Context) {
	q := bq.client.Query(`TRUNCATE TABLE ` + strings.ReplaceAll(bq.table.FullyQualifiedName(), ":", "."))
	q.Priority = bigquery.InteractivePriority
	job, err := q.Run(ctx)
	if err != nil {
		log.Errorf("bq truncate: %v", err)
	}

	// Await timout or job completion
	s, err := job.Wait(ctx)
	switch {
	case err != nil:
		log.Errorf("bq truncate: %v", err)
		return
	case s.Err() != nil:
		log.Errorf("bq truncate: %v", s.Err())
		return
	}
}

func (bq *BigQuery) Init(ctx context.Context) error {
	errorOK := func(err error) bool {
		e, ok := err.(*googleapi.Error)
		if ok && e.Code == 409 {
			// Status code 409 indicates table and dataset combination already exists
			return true
		}
		return false
	}

	defer bq.retryContextConfig.Cancel()

	err := retry.Do(
		bq.retryContextConfig,
		func() error { return createBigQueryTable(ctx, bq.table) },
		errorOK,
	)

	if err != nil {
		return fmt.Errorf("couldn't create bigquery table '%s': %w", bq.table.FullyQualifiedName(), err)
	}

	return nil
}

func createBigQueryTable(ctx context.Context, tableRef *bigquery.Table) error {
	sampleSchema := bigquery.Schema{
		{Name: "Message", Type: bigquery.StringFieldType},
	}

	metaData := &bigquery.TableMetadata{
		Schema:         sampleSchema,
		ExpirationTime: time.Now().AddDate(1, 0, 0), // Table will be automatically deleted in 1 year.
	}

	return tableRef.Create(ctx, metaData)
}
