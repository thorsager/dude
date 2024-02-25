package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/thorsager/dude/metricsandlogging"
	"github.com/thorsager/dude/middleware"
	"github.com/thorsager/dude/persistence"
	"github.com/thorsager/dude/requestid"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const EnvDbPrefix = "DB_"
const EnvUrlSuffix = "_URL"

type Dude struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Phrase string `json:"phrase"`
}

var defaultDataBase = ""

var handler = middleware.Compose(
	func(handlerFunc http.HandlerFunc) http.HandlerFunc {
		return persistence.Middleware(handlerFunc, dbSelector)
	},
	metricsandlogging.Middleware,
	requestid.Middleware,
)

func dbSelector(r *http.Request) string {
	if s := r.Header.Get("X-DB-Name"); s != "" {
		return strings.ToUpper(s)
	}
	return defaultDataBase
}

func readEnvironment() ([]persistence.NamedUrl, error) {
	var nurls []persistence.NamedUrl
	for _, envKv := range os.Environ() {
		if strings.HasPrefix(envKv, EnvDbPrefix) {
			kv := strings.SplitN(envKv, "=", 2)
			if !strings.HasSuffix(kv[0], EnvUrlSuffix) {
				continue
			}
			name := strings.TrimPrefix(kv[0], EnvDbPrefix)
			name = strings.TrimSuffix(name, EnvUrlSuffix)
			u, err := url.Parse(kv[1])
			if err != nil {
				return nurls, err
			}
			nurls = append(nurls, persistence.NamedUrl{Name: strings.ToUpper(name), Url: u})
		}
	}
	return nurls, nil
}

func main() {
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		log.Fatal("Error loading .env file")
	}

	nurls, err := readEnvironment()
	if err != nil {
		panic(err)
	}

	err = persistence.Setup(nurls)
	if err != nil {
		persistence.Close()
		panic(err)
	}
	defer persistence.Close()

	// If only one database is configured, use it as the default
	if len(nurls) == 1 {
		defaultDataBase = nurls[0].Name
	}

	// Register the metrics handler
	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("POST /dude", handler(createDude))
	http.HandleFunc("GET /dude", handler(getAllDudes))
	http.HandleFunc("GET /dude/{id}", handler(getDudeById))
	http.HandleFunc("PUT /dude", handler(updateDude))
	http.HandleFunc("DELETE /dude/{id}", handler(deleteDude))

	log.Printf("Server starting on port 8080")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}

func createDude(w http.ResponseWriter, r *http.Request) {
	var d Dude
	err := json.NewDecoder(r.Body).Decode(&d)
	if err != nil {
		log.Printf("could not decode dude: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		log.Printf("could not prepare statement: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	db := persistence.GetConnection(ctx)

	id := 0
	err = db.QueryRowContext(ctx, "INSERT INTO dudes (name, phrase) VALUES ($1, $2) RETURNING id", d.Name, d.Phrase).Scan(&id)
	if err != nil {
		log.Printf("could not execute statement: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	d.ID = id
	err = json.NewEncoder(w).Encode(d)
	if err != nil {
		log.Printf("could not encode dude: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getDudeById(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx := r.Context()
	db := persistence.GetConnection(ctx)
	var d Dude
	row := db.QueryRowContext(ctx, "SELECT id, name, phrase FROM dudes WHERE id = $1", id)
	err := row.Scan(&d.ID, &d.Name, &d.Phrase)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(d)
	if err != nil {
		log.Printf("could not encode dudes: %v", err)
	}
}

func getAllDudes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	db := persistence.GetConnection(ctx)
	rows, err := db.QueryContext(ctx, "SELECT id, name, phrase FROM dudes ORDER BY id")
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var allDdudes []Dude
	for rows.Next() {
		var d Dude
		err := rows.Scan(&d.ID, &d.Name, &d.Phrase)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		allDdudes = append(allDdudes, d)
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(allDdudes)
	if err != nil {
		log.Printf("could not encode dudes: %v", err)
	}
}

func updateDude(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var d Dude
	err := json.NewDecoder(r.Body).Decode(&d)
	if err != nil {
		log.Printf("could not decode dude: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	db := persistence.GetConnection(ctx)

	result, err := db.ExecContext(ctx, "UPDATE dudes SET name=$2, phrase=$3 WHERE id = $1", d.ID, d.Name, d.Phrase)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = result.RowsAffected()
	if err != nil {
		log.Printf("could not determine rows affected: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusNoContent)
	return
}

func deleteDude(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")
	db := persistence.GetConnection(ctx)
	result, err := db.ExecContext(ctx, "DELETE FROM dudes WHERE id = $1", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	affected, err := result.RowsAffected()
	if err == nil {
		log.Printf("deleted %d dudes", affected)
	} else {
		log.Printf("could not determine rows affected: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
