package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/nais/testapp/pkg/metrics"
	"github.com/nais/testapp/pkg/util"
)

type Database struct {
	client *sql.DB
}

func (db *Database) Name() string {
	return "database"
}

func (db *Database) Test(ctx context.Context, data string) (string, error) {
	err := db.write(ctx, data)
	if err != nil {
		return "", err
	}

	return db.read(ctx)
}

func (db *Database) Cleanup() {
	err := db.client.Close()
	if err != nil {
		log.Errorf("cleanup database: %v", err)
	}
}

func NewDatabaseTest(dbUser, dbPassword, dbName, dbHost string) (*Database, error) {
	err := verifyDbPrerequisites(dbHost, dbPassword)
	if err != nil {
		return nil, err
	}

	client, err := connectToDb(dbUser, dbPassword, dbName, dbHost)
	if err != nil {
		return nil, err
	}

	return &Database{
		client: client,
	}, nil
}

//goland:noinspection SqlNoDataSourceInspection
func (db *Database) Init(ctx context.Context, retries int) error {
	retryCtx, cancel := context.WithTimeout(ctx, 500*time.Duration(retries)*time.Millisecond)
	defer cancel()

	err := util.Retry(retryCtx, func() error {
		stmt := `CREATE TABLE IF NOT EXISTS test (timestamp BIGINT, data VARCHAR(255))`
		_, err := db.client.ExecContext(ctx, stmt)
		return err
	}, func(err error) bool {
		return false
	}, retries)

	if err != nil {
		return fmt.Errorf("failed creating table, error was: %s", err)
	}

	return nil
}

//goland:noinspection SqlNoDataSourceInspection,SqlResolve
func (db *Database) write(ctx context.Context, content string) error {
	// Ensure empty table.
	stmt := `TRUNCATE TABLE test`
	_, err := db.client.ExecContext(ctx, stmt)
	if err != nil {
		return fmt.Errorf("failed to truncate table: %s", err)
	}

	stmt = ` INSERT INTO test (timestamp, data) VALUES ($1, $2)`
	start := time.Now()
	_, err = db.client.ExecContext(ctx, stmt, time.Now().UnixNano(), content)
	if err != nil {
		metrics.DbInsertFailed.Inc()
		return fmt.Errorf("failed inserting to table, error was: %s", err)
	}
	latency := time.Since(start)
	metrics.DbInsert.Set(float64(latency.Nanoseconds()))
	metrics.DbInsertHist.Observe(float64(latency.Nanoseconds()))
	log.Debugf("write to database took %d ms", latency.Milliseconds())
	return nil
}

//goland:noinspection SqlNoDataSourceInspection,SqlResolve
func (db *Database) read(ctx context.Context) (string, error) {
	start := time.Now()
	rows, err := db.client.QueryContext(ctx, "SELECT data FROM test")
	if err != nil {
		metrics.DbReadFailed.Inc()
		return "", fmt.Errorf("could not get rows: %v", err)
	}

	latency := time.Since(start)
	metrics.DbReadHist.Observe(float64(latency.Nanoseconds()))
	metrics.DbRead.Set(float64(latency.Nanoseconds()))
	log.Debugf("read from database took %d ns", latency.Milliseconds())
	defer rows.Close()

	if rows.Next() {
		var data string
		err = rows.Scan(&data)

		if err != nil {
			return "", err
		}

		return data, nil
	} else {
		return "", fmt.Errorf("no rows returned")
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
