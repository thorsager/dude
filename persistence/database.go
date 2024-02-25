package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/dlmiddlecote/sqlstats"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"net/url"
	"strings"
)

type NamedUrl struct {
	Name string
	Url  *url.URL
}

const paramPrefix = "_x-"
const ParamPoolSize = paramPrefix + "poolSize"

type SelectorFunc func(r *http.Request) string

var dbMap map[string]*sql.DB

type dbKey struct{} // key for the context value

func Setup(dbUrls []NamedUrl) error {
	if len(dbUrls) == 0 {
		return fmt.Errorf("no databases specified")
	}

	if dbMap == nil {
		dbMap = make(map[string]*sql.DB)
	}

	for _, nu := range dbUrls {
		if nu.Name == "" {
			return fmt.Errorf("database with empty name")
		}
		if _, ok := dbMap[nu.Name]; ok {
			return fmt.Errorf("database with name %s already exists", nu.Name)
		}
		var err error
		var db *sql.DB

		poolSize := getQueryParamAsInt(nu.Url, ParamPoolSize, 10)
		nu.Url = stripXQueryParam(nu.Url)

		db, err = sql.Open(nu.Url.Scheme, nu.Url.String())
		if err != nil {
			return fmt.Errorf("could not connect to db: %v", err)
		}
		if err = db.Ping(); err != nil {
			return fmt.Errorf("could not ping db: %v", err)
		}

		db.SetMaxOpenConns(poolSize)

		collector := sqlstats.NewStatsCollector(nu.Name, db)
		prometheus.MustRegister(collector)

		dbMap[nu.Name] = db
	}
	return nil
}

func getQueryParamAsInt(u *url.URL, key string, defaultValue int) int {
	if v := u.Query().Get(key); v != "" {
		return defaultValue
	}
	return defaultValue
}

func stripXQueryParam(u *url.URL) *url.URL {
	q := u.Query()
	for k := range q {
		if strings.HasPrefix(k, paramPrefix) {
			q.Del(k)
		}

	}
	u.RawQuery = q.Encode()
	return u
}

func WithConnection(ctx context.Context, dbName string) (context.Context, error) {
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

func Middleware(next http.HandlerFunc, selector SelectorFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbName := selector(r)
		if dbName == "" {
			http.Error(w, "no database specified", http.StatusBadRequest)
			return
		}
		ctx, err := WithConnection(r.Context(), dbName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		next(w, r.WithContext(ctx))
	}
}
