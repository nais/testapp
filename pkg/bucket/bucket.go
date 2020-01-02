package bucket

import (
	"cloud.google.com/go/storage"
	"context"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

func ReadBucketHandler(bucketName, bucketObjectName string) func(w http.ResponseWriter, _ *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		//if err := verifyBucketPrerequisites(bucketName, serviceAccountCredentialsFile); err != nil {
		//	w.WriteHeader(http.StatusInternalServerError)
		//	w.Write([]byte(err.Error()))
		//}

		client, err := storage.NewClient(context.Background())
		if err != nil {
			log.Errorf("error creating storage client: %s", err)
		}

		reader, err := client.Bucket(bucketName).Object(bucketObjectName).NewReader(context.Background())
		defer reader.Close()

		if err != nil {
			log.Errorf("unable to create reader: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := ioutil.ReadAll(reader)
		if err != nil {
			log.Errorf("unable to read from bucket: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(res)
	}

}

func WriteBucketHandler(bucketName, bucketObjectName string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		//if err := verifyBucketPrerequisites(bucketName, serviceAccountCredentialsFile); err != nil {
		//	w.WriteHeader(http.StatusInternalServerError)
		//	_, _ = w.Write([]byte(err.Error()))
		//	return
		//}

		body, err := ioutil.ReadAll(r.Body)
		d := string(body)
		if len(d) > 5 || len(d) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("cannot be empty or more than 5 characters"))
			return
		}
		defer r.Body.Close()

		client, err := storage.NewClient(context.Background())

		if err != nil {
			log.Errorf("error creating storage client: %s", err)
		}

		writer := client.Bucket(bucketName).Object(bucketObjectName).NewWriter(context.Background())
		_, err = writer.Write([]byte(d))

		if err != nil {
			log.Errorf("unable to write to bucket: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err := writer.Close(); err != nil {
			log.Errorf("unable to close bucket writer: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

//func verifyBucketPrerequisites(bucketName, serviceAccountCredentialsFile string) error {
//	if len(bucketName) == 0 {
//		return fmt.Errorf("missing bucket-name")
//	}
//
//	if _, err := os.Stat(serviceAccountCredentialsFile); err != nil {
//		return fmt.Errorf("missing service account credentials")
//	}
//
//	return nil
//}
