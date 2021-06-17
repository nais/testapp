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
)

// Item represents a row item.
type Item struct {
	Message string
}

func ReadBigQueryHandler(projectID, datasetID, tableID string) func(w http.ResponseWriter, _ *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		ctx := context.Background()
		client, err := bigquery.NewClient(ctx, projectID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		defer func(client *bigquery.Client) {
			// Ignore handling this error
			_ = client.Close()
		}(client)

		tableRef := client.Dataset(datasetID).Table(tableID)
		tableRows := tableRef.Read(ctx)
		var row Item
		c := 0
		for {
			err := tableRows.Next(&row)
			if err == iterator.Done {
				break
			}
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(err.Error()))
				return
			}
			c += 1
			log.Infof("Row: %v", row)
		}

		q := client.Query(`TRUNCATE TABLE ` + strings.ReplaceAll(tableRef.FullyQualifiedName(), ":", "."))
		q.Priority = bigquery.InteractivePriority
		job, err := q.Run(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		s, err := job.Wait(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		if s.Err() != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		if c != 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprintf(w, "Test returned incorrect amount of data %v", c)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, row.Message)
	}
}

func createBigQueryTable(ctx context.Context, tableRef *bigquery.Table) error {
	log.Infof("Create new table")
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
	{
		ctx := context.Background()

		client, err := bigquery.NewClient(ctx, projectID)
		if err != nil {
			log.WithError(err).Println("couldn't create bigquery client")
		} else {
			dataset := client.Dataset(datasetID)
			tableRef := dataset.Table(tableID)
			err = createBigQueryTable(ctx, tableRef)
			if err != nil {
				e, ok := err.(*googleapi.Error)
				if !ok || e.Code != 409 {
					log.WithError(err).Println("couldn't create bigquery table")
				}
			}
		}
	}

	return func(w http.ResponseWriter, req *http.Request) {
		ctx := context.Background()
		client, err := bigquery.NewClient(ctx, projectID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		defer func(client *bigquery.Client) {
			// Ignore handling this error
			_ = client.Close()
		}(client)

		b, err := io.ReadAll(req.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		item := Item{
			Message: string(b),
		}

		log.Infof("Inserting row: %v", item)

		dataset := client.Dataset(datasetID)
		tableRef := dataset.Table(tableID)

		q := client.Query("INSERT INTO `" + strings.ReplaceAll(tableRef.FullyQualifiedName(), ":", ".") + "` VALUES (\"" + item.Message + `")`)
		q.Priority = bigquery.InteractivePriority
		job, err := q.Run(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		s, err := job.Wait(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		if s.Err() != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		log.Infof("Finised with request: %s", string(b))
		w.WriteHeader(http.StatusCreated)
	}
}
