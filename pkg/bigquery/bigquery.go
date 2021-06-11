package bigquery

import (
	"cloud.google.com/go/bigquery"
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

func ReadBigQueryHandler(projectID, datasetID string) func(w http.ResponseWriter, _ *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {

		tableID := "dummyTable"
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
		metadata, err := tableRef.Metadata(ctx)
		if err != nil {
			log.Errorf("unable to read metadata from table inn dataset: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(metadata.FullID))

	}
}

func createBigQueryTable(ctx context.Context, datasetHandler *bigquery.Dataset, tableName string) (*bigquery.Table, error){
	tableID := "dummyTable"
	if len(tableName) > 0 {
		tableID = tableName
	}

	sampleSchema := bigquery.Schema{
		{Name: "id", Type: bigquery.StringFieldType},
		{Name: "name", Type: bigquery.StringFieldType},
	}

	metaData := &bigquery.TableMetadata{
		Schema:         sampleSchema,
		ExpirationTime: time.Now().AddDate(1, 0, 0), // Table will be automatically deleted in 1 year.
	}

	tableRef := datasetHandler.Table(tableID)
	if err := tableRef.Create(ctx, metaData); err != nil {
		log.Errorf("failed creating table, error was: %s", err)
		return nil, err
	}

	return tableRef, nil
}

// Item represents a row item.
type Item struct {
	Foo string
	Bar string
}

const specificTimeWhichTimeLibParsesForFormattingPurposes = "2006-01-02T15:04:05Z07:00"

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
		if tableRef == nil {
			tableRef, err = createBigQueryTable(ctx, dataset, tableID)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(err.Error()))
				return
			}
		}

		inserter := tableRef.Inserter()
		items := []*Item{
			{
				Foo: "Hello",
				Bar: fmt.Sprintf(time.Now().UTC().Format(specificTimeWhichTimeLibParsesForFormattingPurposes)),
			},
			{
				Foo: ", world!",
				Bar: fmt.Sprintf(time.Now().UTC().Format(specificTimeWhichTimeLibParsesForFormattingPurposes)),
			},
		}

		err = inserter.Put(ctx, items)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
	}
}
