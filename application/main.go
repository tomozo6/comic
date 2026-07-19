package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/tomozo6/comic/application/internal/catalog"
)

//go:embed media/*
var mediaFiles embed.FS

type identity struct {
	UID   string
	Name  string
	Email string
}

type tokenVerifier interface {
	Verify(context.Context, string) (identity, error)
}

type firebaseVerifier struct{ client *auth.Client }

func (v firebaseVerifier) Verify(ctx context.Context, rawToken string) (identity, error) {
	token, err := v.client.VerifyIDToken(ctx, rawToken)
	if err != nil {
		return identity{}, err
	}
	email, _ := token.Claims["email"].(string)
	name, _ := token.Claims["name"].(string)
	if email == "" {
		return identity{}, errors.New("Firebase ID token does not include an email address")
	}
	return identity{UID: token.UID, Name: name, Email: email}, nil
}

type localMediaSigner struct {
	secret []byte
	ttl    time.Duration
}

func newLocalMediaSigner(secret []byte, ttl time.Duration) localMediaSigner {
	return localMediaSigner{secret: secret, ttl: ttl}
}

func (s localMediaSigner) Sign(key string, now time.Time) (string, error) {
	payload, err := json.Marshal(struct {
		Key       string `json:"key"`
		ExpiresAt int64  `json:"expiresAt"`
	}{Key: key, ExpiresAt: now.Add(s.ttl).Unix()})
	if err != nil {
		return "", err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(encodedPayload))
	return encodedPayload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (s localMediaSigner) Verify(token string, now time.Time) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", errors.New("invalid local media token")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", errors.New("invalid local media token")
	}
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(parts[0]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return "", errors.New("invalid local media token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", errors.New("invalid local media token")
	}
	var data struct {
		Key       string `json:"key"`
		ExpiresAt int64  `json:"expiresAt"`
	}
	if err := json.Unmarshal(payload, &data); err != nil || data.Key == "" || now.Unix() >= data.ExpiresAt {
		return "", errors.New("expired or invalid local media token")
	}
	return data.Key, nil
}

type manga struct {
	ID     string
	Title  string
	Author string
}

type app struct {
	verifier tokenVerifier
	allowed  map[string]struct{}
	signer   localMediaSigner
	db       *sql.DB
}

func (a app) authenticate(w http.ResponseWriter, r *http.Request) (identity, bool) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication is required"})
		return identity{}, false
	}
	user, err := a.verifier.Verify(r.Context(), strings.TrimPrefix(header, "Bearer "))
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid Firebase ID token"})
		return identity{}, false
	}
	if _, ok := a.allowed[strings.ToLower(user.Email)]; !ok {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "this account is not allowed"})
		return identity{}, false
	}
	return user, true
}

func (a app) handleMe(w http.ResponseWriter, r *http.Request) {
	user, ok := a.authenticate(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"uid": user.UID, "name": user.Name, "email": user.Email})
}

func (a app) handleManga(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.authenticate(w, r); !ok {
		return
	}
	type response struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Author string `json:"author"`
	}
	rows, err := a.db.QueryContext(r.Context(), `SELECT id, title, author_name FROM mangas ORDER BY title, id`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not read catalog"})
		return
	}
	defer rows.Close()
	var items []response
	for rows.Next() {
		var item response
		if err := rows.Scan(&item.ID, &item.Title, &item.Author); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not read catalog"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not read catalog"})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a app) handleMangaDetail(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.authenticate(w, r); !ok {
		return
	}
	item, found, err := a.findManga(r.Context(), r.PathValue("mangaID"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not read catalog"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "manga not found"})
		return
	}
	type volumeResponse struct {
		ID     string `json:"id"`
		Number int    `json:"number"`
		Title  string `json:"title"`
	}
	rows, err := a.db.QueryContext(r.Context(), `SELECT id, number, title FROM volumes WHERE manga_id = ? ORDER BY number`, item.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not read catalog"})
		return
	}
	defer rows.Close()
	var volumes []volumeResponse
	for rows.Next() {
		var volume volumeResponse
		if err := rows.Scan(&volume.ID, &volume.Number, &volume.Title); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not read catalog"})
			return
		}
		volumes = append(volumes, volume)
	}
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not read catalog"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": item.ID, "title": item.Title, "author": item.Author, "volumes": volumes})
}

func (a app) handleVolume(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.authenticate(w, r); !ok {
		return
	}
	var item manga
	var volumeID, volumeTitle, pageExtension string
	var volumeNumber, pageCount int
	err := a.db.QueryRowContext(r.Context(), `
SELECT m.id, m.title, m.author_name, v.id, v.number, v.title, v.page_count, v.page_extension
FROM mangas m JOIN volumes v ON v.manga_id = m.id
WHERE m.id = ? AND v.id = ?`, r.PathValue("mangaID"), r.PathValue("volumeID")).Scan(
		&item.ID, &item.Title, &item.Author, &volumeID, &volumeNumber, &volumeTitle, &pageCount, &pageExtension)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "volume not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not read catalog"})
		return
	}
	type pageResponse struct {
		Number   int    `json:"number"`
		ImageURL string `json:"imageUrl"`
	}
	pages := make([]pageResponse, 0, pageCount)
	for number := 1; number <= pageCount; number++ {
		key := fmt.Sprintf("manga/%s/%s/%03d.%s", item.ID, volumeID, number, pageExtension)
		token, err := a.signer.Sign(key, time.Now())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not issue image URL"})
			return
		}
		pages = append(pages, pageResponse{Number: number, ImageURL: "/local-media/" + token})
	}
	writeJSON(w, http.StatusOK, map[string]any{"mangaTitle": item.Title, "volumeNumber": volumeNumber, "volumeTitle": volumeTitle, "pages": pages})
}

func (a app) handleLocalMedia(w http.ResponseWriter, r *http.Request) {
	key, err := a.signer.Verify(r.PathValue("token"), time.Now())
	if err != nil {
		http.NotFound(w, r)
		return
	}
	contents, err := mediaFiles.ReadFile("media/" + key)
	if err != nil {
		contents, err = demoMedia(key)
		if err != nil {
			http.NotFound(w, r)
			return
		}
	}
	if contentType := mime.TypeByExtension(filepath.Ext(key)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	http.ServeContent(w, r, key, time.Time{}, bytes.NewReader(contents))
}

func demoMedia(key string) ([]byte, error) {
	if filepath.Ext(key) != ".svg" {
		return nil, os.ErrNotExist
	}
	base := strings.TrimSuffix(filepath.Base(key), ".svg")
	number, err := strconv.Atoi(base)
	if err != nil || number < 1 || number > 3 {
		return nil, os.ErrNotExist
	}
	return mediaFiles.ReadFile(fmt.Sprintf("media/demo/first-%02d.svg", number))
}

func (a app) findManga(ctx context.Context, id string) (manga, bool, error) {
	var item manga
	err := a.db.QueryRowContext(ctx, `SELECT id, title, author_name FROM mangas WHERE id = ?`, id).Scan(&item.ID, &item.Title, &item.Author)
	if errors.Is(err, sql.ErrNoRows) {
		return manga{}, false, nil
	}
	if err != nil {
		return manga{}, false, err
	}
	return item, true, nil
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func servePage(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join("public", name))
	}
}

func openCatalogForServer(ctx context.Context) (*sql.DB, func(), error) {
	if path := os.Getenv("CATALOG_DB"); path != "" {
		db, err := catalog.OpenReadonly(path)
		return db, func() {}, err
	}
	source := os.Getenv("CATALOG_SOURCE_DIR")
	if source == "" {
		source = "catalog/mangas"
	}
	file, err := os.CreateTemp("", "comic-catalog-*.db")
	if err != nil {
		return nil, nil, err
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		return nil, nil, err
	}
	cleanup := func() { _ = os.Remove(path) }
	if err := catalog.Build(ctx, source, path); err != nil {
		cleanup()
		return nil, nil, err
	}
	db, err := catalog.OpenReadonly(path)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	return db, cleanup, nil
}

func main() {
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	secret := os.Getenv("MEDIA_URL_SIGNING_SECRET")
	allowedEmails := os.Getenv("ALLOWED_EMAILS")
	if projectID == "" || secret == "" || allowedEmails == "" {
		log.Fatal("FIREBASE_PROJECT_ID, ALLOWED_EMAILS, and MEDIA_URL_SIGNING_SECRET must be set")
	}
	allowed := make(map[string]struct{})
	for _, email := range strings.Split(allowedEmails, ",") {
		if normalized := strings.ToLower(strings.TrimSpace(email)); normalized != "" {
			allowed[normalized] = struct{}{}
		}
	}
	if len(allowed) == 0 {
		log.Fatal("ALLOWED_EMAILS must contain at least one email address")
	}

	ctx := context.Background()
	catalogDB, cleanupCatalog, err := openCatalogForServer(ctx)
	if err != nil {
		log.Fatalf("open catalog: %v", err)
	}
	defer catalogDB.Close()
	defer cleanupCatalog()
	firebaseApp, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: projectID})
	if err != nil {
		log.Fatalf("initialize Firebase: %v", err)
	}
	client, err := firebaseApp.Auth(ctx)
	if err != nil {
		log.Fatalf("initialize Firebase Auth client: %v", err)
	}

	a := app{verifier: firebaseVerifier{client: client}, allowed: allowed, signer: newLocalMediaSigner([]byte(secret), 10*time.Minute), db: catalogDB}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	address := ":" + port
	log.Printf("server listening on http://localhost%s", address)
	if err := http.ListenAndServe(address, newRouter(a)); err != nil {
		log.Fatal(err)
	}
}

func newRouter(a app) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/me", a.handleMe)
	mux.HandleFunc("GET /api/manga", a.handleManga)
	mux.HandleFunc("GET /api/manga/{mangaID}", a.handleMangaDetail)
	mux.HandleFunc("GET /api/manga/{mangaID}/volumes/{volumeID}", a.handleVolume)
	mux.HandleFunc("GET /local-media/{token}", a.handleLocalMedia)
	mux.HandleFunc("GET /", servePage("index.html"))
	mux.HandleFunc("GET /library", servePage("library.html"))
	mux.HandleFunc("GET /manga/{mangaID}", servePage("manga.html"))
	mux.HandleFunc("GET /manga/{mangaID}/volumes/{volumeID}", servePage("reader.html"))
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("public/assets"))))
	return mux
}
