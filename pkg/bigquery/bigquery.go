package bigquery

import (
	"context"
	"fmt"
	"io"
	"net/http"
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

func ReadBigQueryHandler(projectID, datasetID, tableID string) func(w http.ResponseWriter, _ *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
		defer cancel()

		client, err := bigquery.NewClient(ctx, projectID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprint(w, err)
			return
		}
		defer func(client *bigquery.Client) {
			// Ignore handling this error
			_ = client.Close()
		}(client)

		tableRef := client.Dataset(datasetID).Table(tableID)
		defer func() {
			// We want this to happen no matter the value of c
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()
			if err := truncateDatabase(ctx, client, tableRef); err != nil {
				log.Errorf("unable to truncate table '%v' after read: %v", tableRef.FullyQualifiedName(), err)
			}
		}()
		start := time.Now()
		tableRows := tableRef.Read(ctx)
		var row Item
		c := 0
		for {
			err := tableRows.Next(&row)
			if err == iterator.Done {
				break
			}
			if err != nil {
				metrics.BqReadFailed.Inc()
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = fmt.Fprint(w, err)
				return
			}
			c += 1
		}
		latency := float64(time.Since(start))
		metrics.BqRead.Set(latency)
		metrics.BqReadHist.Observe(latency)
		log.Debugf("read from big query took %d ns", latency)

		if c != 1 {
			metrics.BqReadFailed.Inc()
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprintf(w, "Test returned incorrect amount of data %v", c)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, row.Message)
	}
}

func truncateDatabase(ctx context.Context, client *bigquery.Client, tableRef *bigquery.Table) error {
	q := client.Query(`TRUNCATE TABLE ` + strings.ReplaceAll(tableRef.FullyQualifiedName(), ":", "."))
	q.Priority = bigquery.InteractivePriority
	job, err := q.Run(ctx)
	if err != nil {
		return err
	}

	// Await timout or job completion
	s, err := job.Wait(ctx)
	switch {
	case err != nil:
		return err
	case s.Err() != nil:
		return s.Err()
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

	if err := tableRef.Create(ctx, metaData); err != nil {
		log.Errorf("failed creating table, error was: %s", err)
		return err
	}

	return nil
}

func WriteBigQueryHandler(projectID, datasetID, tableID string) func(w http.ResponseWriter, _ *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		client, err := bigquery.NewClient(ctx, projectID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprint(w, err)
			return
		}
		defer func(client *bigquery.Client) {
			// Ignore handling this error
			_ = client.Close()
		}(client)

		// Get input from POST request body
		b, err := io.ReadAll(req.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprint(w, err)
			return
		}

		// Prepare query
		item := Item{
			Message: string(b),
		}
		tableRef := client.Dataset(datasetID).Table(tableID)
		q := client.Query("INSERT INTO `" + strings.ReplaceAll(tableRef.FullyQualifiedName(), ":", ".") + "` VALUES (\"" + item.Message + `")`)
		q.Priority = bigquery.InteractivePriority

		// Execute query
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		start := time.Now()
		job, err := q.Run(ctx)
		if err != nil {
			metrics.BqInsert.Inc()
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprint(w, err)
			return
		}

		// Await timout or job completion
		s, err := job.Wait(ctx)
		if err != nil || s.Err() != nil {
			if err == nil {
				err = s.Err()
			}
			metrics.BqInsertFailed.Inc()
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprint(w, err)
			return
		}
		latency := float64(time.Since(start))
		metrics.BqInsert.Set(latency)
		metrics.BqInsertHist.Observe(latency)
		log.Debugf("write to big query took %d ns", latency)
		w.WriteHeader(http.StatusCreated)
	}
}

func CreateDatasetAndTable(projectID string, datasetID string, tableID string) error {
	ctx := context.Background()

	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("couldn't create bigquery client in project-id '%v': %w", projectID, err)
	}

	for tries := 0; tries < 10; tries++ {
		tableRef := client.Dataset(datasetID).Table(tableID)
		err = createBigQueryTable(ctx, tableRef)
		if err != nil {
			e, ok := err.(*googleapi.Error)
			if ok && e.Code == 409 {
				// Status code 409 indicates table and dataset combination already exists
				return nil
			}

			time.Sleep(500 * time.Duration(tries+1) * time.Millisecond)
			continue
		}
		break
	}

	if err != nil {
		return fmt.Errorf("couldn't create bigquery table '%v.%v.%v': %w", projectID, datasetID, tableID, err)
	}

	return nil
}
