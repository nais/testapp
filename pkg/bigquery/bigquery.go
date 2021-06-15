package bigquery

import (
	"cloud.google.com/go/bigquery"
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"io"
	"net/http"
	"time"
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

		tableRows := client.Dataset(datasetID).Table(tableID).Read(ctx)
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

		dataset := client.Dataset(datasetID)
		tableRef := dataset.Table(tableID)

		// Delete potentially existing table
		log.Infof("Delete bigquery table %v", tableRef.TableID)
		err = tableRef.Delete(ctx)
		if err != nil {
			e, ok := err.(*googleapi.Error)
			if !ok || e.Code != 404 {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(err.Error()))
				return
			}
		}
		err = createBigQueryTable(ctx, tableRef)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		b, err := io.ReadAll(req.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		items := []*Item{
			{
				Message: string(b),
			},
		}

		log.Infof("Inserting row: %v", items)
		inserter := tableRef.Inserter()
		err = inserter.Put(ctx, items)
		if err != nil {
			log.Errorf("insert failed %v", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		log.Infof("Finised with request: %s", string(b))
		w.WriteHeader(http.StatusCreated)
	}
}
