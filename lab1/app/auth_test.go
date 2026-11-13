package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func setup() http.Handler {
	auth = authConfig{secret: []byte("0123456789abcdef-test"), username: "admin", password: "pw"}
	mux := http.NewServeMux()
	registerAuth(mux)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("GET /playlists", func(w http.ResponseWriter, r *http.Request) {})
	return requireAuth(mux)
}

func do(h http.Handler, method, path, body, authz string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if authz != "" {
		req.Header.Set("Authorization", authz)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAuthFlow(t *testing.T) {
	h := setup()
	if c := do(h, "GET", "/health", "", "").Code; c != 200 {
		t.Fatalf("health %d", c)
	}
	if c := do(h, "GET", "/playlists", "", "").Code; c != 401 {
		t.Fatalf("no token %d", c)
	}
	if c := do(h, "GET", "/playlists", "", "Bearer garbage").Code; c != 401 {
		t.Fatalf("garbage %d", c)
	}
	if c := do(h, "POST", "/login", `{"username":"admin","password":"bad"}`, "").Code; c != 401 {
		t.Fatalf("bad login %d", c)
	}
	rec := do(h, "POST", "/login", `{"username":"admin","password":"pw"}`, "")
	if rec.Code != 200 {
		t.Fatalf("login %d", rec.Code)
	}
	body := rec.Body.String()
	i := strings.Index(body, `"token":"`) + 9
	tok := body[i : i+strings.Index(body[i:], `"`)]
	if c := do(h, "GET", "/playlists", "", "Bearer "+tok).Code; c != 200 {
		t.Fatalf("with token %d", c)
	}
	if c := do(h, "GET", "/playlists", "", "Bearer "+tok[:len(tok)-2]+"xx").Code; c != 401 {
		t.Fatalf("tampered %d", c)
	}
	none := b64.EncodeToString([]byte(`{"alg":"none"}`)) + "." + strings.Split(tok, ".")[1] + "."
	if c := do(h, "GET", "/playlists", "", "Bearer "+none).Code; c != 401 {
		t.Fatalf("alg none %d", c)
	}
}

func TestExpired(t *testing.T) {
	setup()
	tok, _ := issueToken("admin", time.Now().Add(-2*time.Hour))
	if verifyToken(tok, time.Now()) == nil {
		t.Fatal("expired token accepted")
	}
}
