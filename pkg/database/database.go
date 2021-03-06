package database

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/nais/testapp/pkg/metrics"
	log "github.com/sirupsen/logrus"
)

func WriteDatabaseHandler(dbUser, dbPassword, dbName, dbHost string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := verifyDbPrerequisites(dbHost, dbPassword); err != nil {
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

		db, err := connectToDb(dbUser, dbPassword, dbName, dbHost)
		defer db.Close()
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
		start := time.Now()
		_, err = db.Exec(stmt, time.Now().UnixNano(), d)
		if err != nil {
			metrics.DbInsertFailed.Inc()
			log.Errorf("failed inserting to table, error was: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		latency := float64(time.Since(start).Nanoseconds())
		metrics.DbInsert.Set(latency)
		metrics.DbInsertHist.Observe(latency)
		log.Debugf("write to database took %d ns", latency)

		w.WriteHeader(http.StatusCreated)
	}
}

func ReadDatabaseHandler(dbUser, dbPassword, dbName, dbHost string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := verifyDbPrerequisites(dbHost, dbPassword); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		db, err := connectToDb(dbUser, dbPassword, dbName, dbHost)
		defer db.Close()

		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		start := time.Now()
		rows, err := db.Query("SELECT data FROM test")
		if err != nil {
			metrics.DbReadFailed.Inc()
			log.Errorf("could not get rows: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		latency := float64(time.Since(start).Nanoseconds())
		metrics.DbReadHist.Observe(latency)
		metrics.DbRead.Set(latency)
		log.Debugf("read from database took %d ns", latency)
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
	}
}

func verifyDbPrerequisites(dbHost, dbPassword string) error {
	if len(dbHost) == 0 || len(dbPassword) == 0 {
		return fmt.Errorf("missing required database config")
	}

	return nil
}

func connectToDb(dbUser, dbPassword, dbName, dbHost string) (*sql.DB, error) {
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
