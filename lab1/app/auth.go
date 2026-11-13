package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const tokenLifetime = time.Hour

var publicPaths = map[string]bool{
	"/health":       true,
	"/login":        true,
	"/docs":         true,
	"/openapi.yaml": true,
}

type authConfig struct {
	secret   []byte
	username string
	password string
}

var auth authConfig

var b64 = base64.RawURLEncoding

func loadAuthConfig() {
	auth.secret = []byte(os.Getenv("JWT_SECRET"))
	auth.username = os.Getenv("ADMIN_USERNAME")
	if auth.username == "" {
		auth.username = "admin"
	}
	auth.password = os.Getenv("ADMIN_PASSWORD")
	if len(auth.secret) < 16 || auth.password == "" {
		log.Fatal("JWT_SECRET (min 16 chars) and ADMIN_PASSWORD must be set")
	}
}

func sign(data string) string {
	mac := hmac.New(sha256.New, auth.secret)
	mac.Write([]byte(data))
	return b64.EncodeToString(mac.Sum(nil))
}

func issueToken(subject string, now time.Time) (string, error) {
	header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	claims, err := json.Marshal(map[string]any{
		"sub": subject,
		"iat": now.Unix(),
		"exp": now.Add(tokenLifetime).Unix(),
	})
	if err != nil {
		return "", err
	}
	unsigned := b64.EncodeToString(header) + "." + b64.EncodeToString(claims)
	return unsigned + "." + sign(unsigned), nil
}

func verifyToken(token string, now time.Time) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return errors.New("malformed token")
	}
	headerJSON, err := b64.DecodeString(parts[0])
	if err != nil {
		return errors.New("malformed header")
	}
	var header struct {
		Alg string `json:"alg"`
	}
	if json.Unmarshal(headerJSON, &header) != nil || header.Alg != "HS256" {
		return errors.New("unsupported algorithm") // also rejects alg=none
	}
	expected := sign(parts[0] + "." + parts[1])
	if subtle.ConstantTimeCompare([]byte(expected), []byte(parts[2])) != 1 {
		return errors.New("bad signature")
	}
	claimsJSON, err := b64.DecodeString(parts[1])
	if err != nil {
		return errors.New("malformed claims")
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if json.Unmarshal(claimsJSON, &claims) != nil || claims.Exp == 0 {
		return errors.New("malformed claims")
	}
	if now.Unix() >= claims.Exp {
		return errors.New("token expired")
	}
	return nil
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	if status == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", `Bearer realm="track-catalog"`)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if publicPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}
		header := r.Header.Get("Authorization")
		token, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || token == "" {
			writeJSONError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		if err := verifyToken(token, time.Now()); err != nil {
			writeJSONError(w, http.StatusUnauthorized, "invalid token: "+err.Error())
			return
		}
		next.ServeHTTP(w, r)
	})
}

func registerAuth(mux *http.ServeMux) {
	mux.HandleFunc("POST /login", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&in); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		userOK := subtle.ConstantTimeCompare([]byte(in.Username), []byte(auth.username)) == 1
		passOK := subtle.ConstantTimeCompare([]byte(in.Password), []byte(auth.password)) == 1
		if !userOK || !passOK {
			writeJSONError(w, http.StatusUnauthorized, "wrong username or password")
			return
		}
		token, err := issueToken(in.Username, time.Now())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "cannot issue token")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":      token,
			"token_type": "Bearer",
			"expires_in": int(tokenLifetime.Seconds()),
		})
	})
}
