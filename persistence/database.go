package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/dlmiddlecote/sqlstats"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
)

type SelectorFunc func(r *http.Request) string

var onlyDb *sql.DB
var dbMap map[string]*sql.DB

type dbKey struct{} // key for the context value

func Setup(dbUrls []NamedUrl) error {
	if len(dbUrls) == 0 {
		return fmt.Errorf("no databases specified")
	}

	if dbMap == nil {
		dbMap = make(map[string]*sql.DB, len(dbUrls))
	}

	for _, nu := range dbUrls {
		if nu.Name == "" {
			return fmt.Errorf("database with empty name")
		}
		if _, ok := dbMap[nu.Name]; ok {
			return fmt.Errorf("database with name %s already exists", nu.Name)
		}
		err := Migrate(nu.StrippedUrl().String())
		if err != nil {
			return fmt.Errorf("could not migrate db: %v", err)
		}
		db, err := createAndConfigurePool(nu)
		if err != nil {
			return err
		}
		dbMap[nu.Name] = db
	}

	if len(dbMap) == 1 {
		onlyDb = dbMap[dbUrls[0].Name] // select first and only db
	} else {
		onlyDb = nil // if Setup() is called multiple times, onlyDb is nil
	}

	return nil
}

func createAndConfigurePool(nu NamedUrl) (*sql.DB, error) {
	// handle custom _x- prefixed parameters
	poolSize := getQueryParamAsInt(nu.Url, ParamPoolSize, 10)

	db, err := sql.Open(nu.Url.Scheme, nu.StrippedUrl().String())
	if err != nil {
		return nil, fmt.Errorf("could not connect to db: %v", err)
	}
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("could not ping db: %v", err)
	}

	db.SetMaxOpenConns(poolSize)

	collector := sqlstats.NewStatsCollector(nu.Name, db)
	prometheus.MustRegister(collector)
	return db, nil
}

func withConnection(ctx context.Context, dbName string) (context.Context, error) {
	if onlyDb != nil {
		return context.WithValue(ctx, dbKey{}, onlyDb), nil
	}

	if db, ok := dbMap[dbName]; !ok {
		return nil, fmt.Errorf("database with name %s does not exist", dbName)
	} else {
		return context.WithValue(ctx, dbKey{}, db), nil
	}
}

func GetConnection(ctx context.Context) *sql.DB {
	return ctx.Value(dbKey{}).(*sql.DB)
}

func Close() {
	if dbMap != nil {
		for _, db := range dbMap {
			_ = db.Close()
		}
	}
}

func Middleware(selector SelectorFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			dbName := selector(r)
			if dbName == "" {
				http.Error(w, "no database specified", http.StatusBadRequest)
				return
			}
			ctx, err := withConnection(r.Context(), dbName)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
