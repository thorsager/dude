package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/dlmiddlecote/sqlstats"
	"github.com/prometheus/client_golang/prometheus"
	"net/url"
)

var db *sql.DB

type dbKeyType int // key for the context value
const dbKey dbKeyType = 0

func Setup(dbUrl *url.URL) error {
	var err error
	db, err = sql.Open(dbUrl.Scheme, dbUrl.String())
	if err != nil {
		return fmt.Errorf("could not connect to db: %v", err)
	}
	if err = db.Ping(); err != nil {
		return fmt.Errorf("could not ping db: %v", err)
	}

	db.SetMaxOpenConns(2)
	// Create a new collector, the name will be used as a label on the metrics
	collector := sqlstats.NewStatsCollector("postgres", db)
	// Register it with Prometheus
	prometheus.MustRegister(collector)
	return nil
}

func WithConnection(ctx context.Context) context.Context {
	return context.WithValue(ctx, dbKey, db)
}

func GetConnection(ctx context.Context) *sql.DB {
	return ctx.Value(dbKey).(*sql.DB)
}

func Close() {
	if db != nil {
		_ = db.Close()
	}
}
