package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const schema = `
CREATE TABLE IF NOT EXISTS playlists (
	id   BIGSERIAL PRIMARY KEY,
	name TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS tracks (
	id           BIGSERIAL PRIMARY KEY,
	title        TEXT NOT NULL,
	artist       TEXT NOT NULL,
	duration_sec INT  NOT NULL CHECK (duration_sec > 0),
	playlist_id  BIGINT NOT NULL REFERENCES playlists(id) ON DELETE CASCADE
);`

type Playlist struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Track struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	DurationSec int    `json:"duration_sec"`
	PlaylistID  int64  `json:"playlist_id"`
}

type app struct{ db *pgxpool.Pool }

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := connect(ctx, dsn)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, schema); err != nil {
		log.Fatalf("schema: %v", err)
	}

	a := &app{db: pool}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.health)
	mux.HandleFunc("GET /playlists", a.listPlaylists)
	mux.HandleFunc("POST /playlists", a.createPlaylist)
	mux.HandleFunc("GET /playlists/{id}/tracks", a.listPlaylistTracks)
	mux.HandleFunc("GET /tracks", a.listTracks)
	mux.HandleFunc("POST /tracks", a.createTrack)
	mux.HandleFunc("GET /tracks/{id}", a.getTrack)
	mux.HandleFunc("PUT /tracks/{id}", a.updateTrack)
	mux.HandleFunc("DELETE /tracks/{id}", a.deleteTrack)

	registerDocs(mux)

	srv := &http.Server{Addr: ":" + port, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("listening on :%s", port)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	var lastErr error
	for i := 0; i < 10; i++ {
		pool, err := pgxpool.New(ctx, dsn)
		if err == nil {
			if err = pool.Ping(ctx); err == nil {
				return pool, nil
			}
			pool.Close()
		}
		lastErr = err
		log.Printf("db not ready (%d/10): %v", i+1, err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
	return nil, lastErr
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func pathID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return id, err == nil && id > 0
}

func (t *Track) valid() bool {
	return t.Title != "" && t.Artist != "" && t.DurationSec > 0 && t.PlaylistID > 0
}

func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func (a *app) queryTracks(r *http.Request, sql string, args ...any) ([]Track, error) {
	rows, err := a.db.Query(r.Context(), sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tracks := []Track{}
	for rows.Next() {
		var t Track
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.DurationSec, &t.PlaylistID); err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

const trackCols = `id, title, artist, duration_sec, playlist_id`

func (a *app) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.db.Ping(ctx); err != nil {
		writeErr(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *app) listPlaylists(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(r.Context(), `SELECT id, name FROM playlists ORDER BY id`)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()
	out := []Playlist{}
	for rows.Next() {
		var p Playlist
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			writeErr(w, http.StatusInternalServerError, "scan failed")
			return
		}
		out = append(out, p)
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *app) createPlaylist(w http.ResponseWriter, r *http.Request) {
	var p Playlist
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil || p.Name == "" {
		writeErr(w, http.StatusBadRequest, "name is required")
		return
	}
	err := a.db.QueryRow(r.Context(), `INSERT INTO playlists (name) VALUES ($1) RETURNING id`, p.Name).Scan(&p.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "insert failed")
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (a *app) listPlaylistTracks(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	tracks, err := a.queryTracks(r, `SELECT `+trackCols+` FROM tracks WHERE playlist_id = $1 ORDER BY id`, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "query failed")
		return
	}
	writeJSON(w, http.StatusOK, tracks)
}

func (a *app) listTracks(w http.ResponseWriter, r *http.Request) {
	tracks, err := a.queryTracks(r, `SELECT `+trackCols+` FROM tracks ORDER BY id`)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "query failed")
		return
	}
	writeJSON(w, http.StatusOK, tracks)
}

func (a *app) createTrack(w http.ResponseWriter, r *http.Request) {
	var t Track
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil || !t.valid() {
		writeErr(w, http.StatusBadRequest, "title, artist, duration_sec>0 and playlist_id are required")
		return
	}
	err := a.db.QueryRow(r.Context(),
		`INSERT INTO tracks (title, artist, duration_sec, playlist_id) VALUES ($1,$2,$3,$4) RETURNING id`,
		t.Title, t.Artist, t.DurationSec, t.PlaylistID).Scan(&t.ID)
	if err != nil {
		if isFKViolation(err) {
			writeErr(w, http.StatusBadRequest, "playlist not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, "insert failed")
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (a *app) getTrack(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var t Track
	err := a.db.QueryRow(r.Context(), `SELECT `+trackCols+` FROM tracks WHERE id = $1`, id).
		Scan(&t.ID, &t.Title, &t.Artist, &t.DurationSec, &t.PlaylistID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeErr(w, http.StatusNotFound, "track not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "query failed")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (a *app) updateTrack(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var t Track
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil || !t.valid() {
		writeErr(w, http.StatusBadRequest, "title, artist, duration_sec>0 and playlist_id are required")
		return
	}
	tag, err := a.db.Exec(r.Context(),
		`UPDATE tracks SET title=$1, artist=$2, duration_sec=$3, playlist_id=$4 WHERE id=$5`,
		t.Title, t.Artist, t.DurationSec, t.PlaylistID, id)
	if err != nil {
		if isFKViolation(err) {
			writeErr(w, http.StatusBadRequest, "playlist not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, "update failed")
		return
	}
	if tag.RowsAffected() == 0 {
		writeErr(w, http.StatusNotFound, "track not found")
		return
	}
	t.ID = id
	writeJSON(w, http.StatusOK, t)
}

func (a *app) deleteTrack(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	tag, err := a.db.Exec(r.Context(), `DELETE FROM tracks WHERE id = $1`, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "delete failed")
		return
	}
	if tag.RowsAffected() == 0 {
		writeErr(w, http.StatusNotFound, "track not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
