package dude

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/thorsager/dude/persistence"
	"github.com/thorsager/dude/requestid"
	"log"
	"net/http"
)

type Dude struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Phrase string `json:"phrase"`
}

func Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rid := requestid.GetID(ctx)
	var d Dude
	err := json.NewDecoder(r.Body).Decode(&d)
	if err != nil {
		log.Printf("[%s] could not decode dude: %v", rid, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		log.Printf("[%s] could not prepare statement: %v", rid, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	db := persistence.GetConnection(ctx)

	id := 0
	err = db.QueryRowContext(ctx, "INSERT INTO dudes (name, phrase) VALUES ($1, $2) RETURNING id", d.Name, d.Phrase).Scan(&id)
	if err != nil {
		log.Printf("[%s] could not execute statement: %v", rid, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	d.ID = id
	err = json.NewEncoder(w).Encode(d)
	if err != nil {
		log.Printf("[%s] could not encode dude: %v", rid, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func GetById(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx := r.Context()
	rid := requestid.GetID(ctx)
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
		log.Printf("[%s] could not encode dudes: %v", rid, err)
	}
}

func GetAll(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rid := requestid.GetID(ctx)
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

	var alldudes = make([]Dude, 0)
	for rows.Next() {
		var d Dude
		err := rows.Scan(&d.ID, &d.Name, &d.Phrase)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		alldudes = append(alldudes, d)
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(alldudes)
	if err != nil {
		log.Printf("[%s] could not encode dudes: %v", rid, err)
	}
}

func Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rid := requestid.GetID(ctx)
	var d Dude
	err := json.NewDecoder(r.Body).Decode(&d)
	if err != nil {
		log.Printf("[%s] could not decode dude: %v", rid, err)
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

	affected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[%s] could not determine rows affected: %v", rid, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if affected == 0 {
		log.Printf("[%s] no rows affected", rid)
	}
	w.WriteHeader(http.StatusNoContent)
	return
}

func Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rid := requestid.GetID(ctx)
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
	if err != nil {
		log.Printf("[%s] could not determine rows affected: %v", rid, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		log.Printf("[%s] no rows affected", rid)
	}
	w.WriteHeader(http.StatusNoContent)
}
