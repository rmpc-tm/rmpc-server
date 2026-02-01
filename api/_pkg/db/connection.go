package db

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/lib/pq"

	"rmpc-server/api/_pkg/config"
)

var (
	dbInstance *sql.DB
	dbOnce     sync.Once
	dbErr      error
)

func GetDB() (*sql.DB, error) {
	dbOnce.Do(func() {
		dsn := config.Env.DatabaseURL
		if dsn == "" {
			dbErr = fmt.Errorf("DATABASE_URL environment variable is not set")
			return
		}

		dbInstance, dbErr = sql.Open("postgres", dsn)
		if dbErr != nil {
			return
		}

		dbInstance.SetMaxOpenConns(10)
		dbInstance.SetMaxIdleConns(5)
		dbInstance.SetConnMaxLifetime(5 * time.Minute)
	})

	return dbInstance, dbErr
}
