package main

import (
	"cloud.google.com/go/storage"
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/nais/testapp/pkg/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"google.golang.org/api/option"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	bindAddr                      string
	pingResponse                  string
	connectURL                    string
	gracefulShutdownPeriodSeconds int
	bucketName                    string
	serviceAccountCredentialsFile string
	bucketObjectName              string
	dbUser                        string
	dbPassword                    string
	dbHost                        string
	dbName                        string
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})

	flag.StringVar(&bindAddr, "bind-address", ":8080", "ip:port where http requests are served")
	flag.StringVar(&pingResponse, "ping-response", "pong\n", "what to respond when pinged")
	flag.StringVar(&bucketName, "bucket-name", os.Getenv("BUCKET_NAME"), "name of bucket used with /{read,write}bucket")
	flag.StringVar(&bucketObjectName, "bucket-object-name", "test", "name of bucket object used with /{read,write}bucket")
	flag.StringVar(&serviceAccountCredentialsFile, "service-account-credentials-file", "/var/run/secrets/testapp-serviceaccount.json", "path to service account credentials file")
	flag.StringVar(&connectURL, "connect-url", "https://google.com", "URL to connect to with /connect")
	flag.StringVar(&dbName, "db-name", getEnv("DB_NAME", "sqldatabase"), "database name")
	flag.StringVar(&dbUser, "db-user", getEnv("DB_USER", "sqluser"), "database username")
	flag.StringVar(&dbPassword, "db-password", os.Getenv("DB_PASSWORD"), "database password")
	flag.StringVar(&dbHost, "db-hostname", os.Getenv("DB_HOST"), "database hostname")
	flag.IntVar(&gracefulShutdownPeriodSeconds, "graceful-shutdown-wait", 0, "when receiving interrupt signal, it will wait this amount of seconds before shutting down server")
	flag.Parse()
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGTERM, syscall.SIGINT)
	hostname, _ := os.Hostname()

	r := mux.NewRouter()

	r.Handle("/metrics", promhttp.Handler())

	r.HandleFunc("/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, pingResponse)
	})

	r.HandleFunc("/version", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "%s (rev: %s)", version.Version, version.Revision)
	})

	r.HandleFunc("/hostname", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, hostname)
	})

	r.HandleFunc("/env", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, os.Environ())
	})

	r.HandleFunc("/log", func(w http.ResponseWriter, _ *http.Request) {
		log.Info("this is a log statement from testapp")
		w.WriteHeader(http.StatusOK)
	})

	r.HandleFunc("/logerror", func(w http.ResponseWriter, _ *http.Request) {
		log.Error("this is a error log statement from testapp")
		w.WriteHeader(http.StatusOK)
	})

	r.HandleFunc("/connect", func(w http.ResponseWriter, _ *http.Request) {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

		resp, err := http.Get(connectURL)
		if err != nil {
			log.Error("error performing http get with url", connectURL, err)
			_, _ = fmt.Fprintf(w, "error performing http get")
			return
		}

		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Error("error reading response body", err)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprint(w, "error reading response body")
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "HTTP status: %d, body:\n%s", resp.StatusCode, string(b))
	})

	r.HandleFunc("/readbucket", func(w http.ResponseWriter, _ *http.Request) {
		if err := verifyBucketPrerequisites(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}

		client, err := storage.NewClient(context.Background(), option.WithCredentialsFile(serviceAccountCredentialsFile))
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
	})

	r.HandleFunc("/writebucket", func(w http.ResponseWriter, r *http.Request) {
		if err := verifyBucketPrerequisites(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		d := string(body)
		if len(d) > 5 || len(d) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("cannot be empty or more than 5 characters"))
			return
		}
		defer r.Body.Close()

		client, err := storage.NewClient(context.Background(), option.WithCredentialsFile(serviceAccountCredentialsFile))

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
	}).Methods(http.MethodPost)

	r.HandleFunc("/writedb", func(w http.ResponseWriter, r *http.Request) {
		if err := verifyDbPrerequisites(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		d := string(body)
		if len(d) > 5 || len(d) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("cannot be empty or more than 5 characters"))
			return
		}

		defer r.Body.Close()

		db, err := connectToDb()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		stmt := `CREATE TABLE IF NOT EXISTS test (
                        timestamp  BIGINT,
                        data     VARCHAR(255)
                )`

		_, err = db.Exec(stmt)

		if err != nil {
			log.Errorf("failed creating table, error was: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		// Ensure empty table.
		stmt = `TRUNCATE TABLE test`
		_, err = db.Exec(stmt)
		if err != nil {
			log.Errorf("failed to truncate table: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		stmt = ` INSERT INTO test (timestamp, data) VALUES ($1, $2)`
		_, err = db.Exec(stmt, time.Now().UnixNano(), d)
		if err != nil {
			log.Errorf("failed inserting to table, error was: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusCreated)
	}).Methods(http.MethodPost)

	r.HandleFunc("/readdb", func(w http.ResponseWriter, r *http.Request) {
		if err := verifyDbPrerequisites(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		db, err := connectToDb()

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		rows, err := db.Query("SELECT data FROM test")

		if err != nil {
			log.Errorf("could not get rows: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		defer rows.Close()

		if rows.Next() {
			row := make([]byte, 10)
			err = rows.Scan(&row)

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(err.Error()))
				return
			}

			log.Infof("%s", row)

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(row)
		} else {
			w.WriteHeader(http.StatusNoContent)
			return
		}
	})

	log.Println("running @", bindAddr)
	server := &http.Server{Addr: bindAddr, Handler: r}

	go func() {
		log.Fatal(server.ListenAndServe())
	}()

	<-interrupt

	log.Printf("allowing %d seconds to shut down gracefully", gracefulShutdownPeriodSeconds)
	time.Sleep(time.Duration(gracefulShutdownPeriodSeconds) * time.Duration(time.Second))
	log.Print("shutting down")

	_ = server.Shutdown(context.Background())
}

func verifyDbPrerequisites() error {
	if len(dbHost) == 0 || len(dbPassword) == 0 {
		return fmt.Errorf("missing required database config")
	}

	return nil
}
func verifyBucketPrerequisites() error {
	if len(bucketName) == 0 {
		return fmt.Errorf("missing bucket-name")
	}

	if _, err := os.Stat(serviceAccountCredentialsFile); err != nil {
		return fmt.Errorf("missing service account credentials")
	}

	return nil
}

func connectToDb() (*sql.DB, error) {
	postgresConnection := fmt.Sprintf("user=%s password=%s dbname=%s host=%s sslmode=disable",
		dbUser,
		dbPassword,
		dbName,
		dbHost)

	db, err := sql.Open("postgres", postgresConnection)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database, error was: %s", err)
	}

	return db, nil
}
