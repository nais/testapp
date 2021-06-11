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

func validateRow(row Item, currentTime time.Time) (bool, error) {
	rowCreationTimestamp, err := time.Parse(specificTimeWhichTimeLibParsesForFormattingPurposes, row.Bar)
	if err != nil {
		log.Errorf("unable to parse timestamp in row '%v': %v", row, err)
		return false, err
	}

	return row.Foo == "Hello, world!" && currentTime.After(rowCreationTimestamp), nil
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

		tableRows := client.Dataset(datasetID).Table(tableID).Read(ctx)
		currentTime := time.Now().UTC()
		for {
			var row Item
			err := tableRows.Next(&row)
			if err == iterator.Done {
				break
			} else if err != nil {
				log.Errorf("error iterating through results: %v", err)
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(err.Error()))
				return
			}

			validRow, err := validateRow(row, currentTime)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(err.Error()))
				return
			}

			if validRow != true {
				w.WriteHeader(http.StatusInternalServerError)
				log.Infof("row was not considered valid: %v", row)
				return
			}
		}
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
				Foo: "Hello, world!",
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
