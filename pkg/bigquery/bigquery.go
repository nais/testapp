package bigquery

import (
	"cloud.google.com/go/bigquery"
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
	"net/http"
	"time"
)

// Item represents a row item.
type Item struct {
	Foo string
	Bar string
}

const specificTimeWhichTimeLibParsesForFormattingPurposes = "2006-01-02T15:04:05Z07:00"

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

		tableRows := client.Dataset(datasetID).Table(tableID).Read(ctx)
		var row []bigquery.Value
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
			log.Infof("Row: %v", row)
		}
		w.WriteHeader(http.StatusOK)
	}
}

func createBigQueryTable(ctx context.Context, tableRef *bigquery.Table) error {

	log.Infof("Create new table")
	sampleSchema := bigquery.Schema{
		{Name: "Foo", Type: bigquery.StringFieldType},
		{Name: "Bar", Type: bigquery.StringFieldType},
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

		dataset := client.Dataset(datasetID)
		tableRef := dataset.Table(tableID)

		// Check if table exists
		_, err = tableRef.Metadata(ctx)
		if err != nil {
			err = createBigQueryTable(ctx, tableRef)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(err.Error()))
				return
			}
		} else {
			// Recursively delete table
			log.Infof("Delete bigquery table %v", tableRef.TableID)
			err := tableRef.Delete(ctx)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(err.Error()))
				return
			}
			err = createBigQueryTable(ctx, tableRef)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(err.Error()))
				return
			}
		}
		inserter := tableRef.Inserter()
		items := []*Item{
			{
				Foo: "Hello, world!",
				Bar: fmt.Sprintf(time.Now().UTC().Format(specificTimeWhichTimeLibParsesForFormattingPurposes)),
			},
		}
		err = inserter.Put(ctx, items)
		if err != nil {
			log.Errorf("insert failed %v", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		log.Infof("Inserting rows")
		w.WriteHeader(http.StatusCreated)

	}
}
