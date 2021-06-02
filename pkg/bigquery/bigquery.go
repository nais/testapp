package bigquery

import (
	"cloud.google.com/go/bigquery"
	"context"
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
		defer client.Close()
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

func WriteBigQueryHandler(projectID, datasetID string) func(w http.ResponseWriter, _ *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {

		tableID := "dummyTable"
		ctx := context.Background()

		client, err := bigquery.NewClient(ctx, projectID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		defer client.Close()

		sampleSchema := bigquery.Schema{
			{Name: "id", Type: bigquery.StringFieldType},
			{Name: "name", Type: bigquery.StringFieldType},
		}

		metaData := &bigquery.TableMetadata{
			Schema:         sampleSchema,
			ExpirationTime: time.Now().AddDate(1, 0, 0), // Table will be automatically deleted in 1 year.
		}
		dataset := client.Dataset(datasetID)
		tableRef := dataset.Table(tableID)
		if err := tableRef.Create(ctx, metaData); err != nil {
			log.Errorf("failed creating table, error was: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

	}

}
