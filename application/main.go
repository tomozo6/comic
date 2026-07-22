package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/tomozo6/manga/application/internal/catalog"
	"google.golang.org/api/iamcredentials/v1"
)

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

const (
	pageURLBatchSize               = 8
	mangaImageBucket               = "tomozo-manga-images"
	mangaImageSignerServiceAccount = "manga-media-signer@tomozo6.iam.gserviceaccount.com"
	mediaURLTTL                    = time.Hour
)

type gcsMediaSigner struct {
	signBytes func(context.Context, []byte) ([]byte, error)
}

func newGCSMediaSigner(ctx context.Context) (gcsMediaSigner, error) {
	service, err := iamcredentials.NewService(ctx)
	if err != nil {
		return gcsMediaSigner{}, fmt.Errorf("create IAM Credentials client: %w", err)
	}
	return gcsMediaSigner{
		signBytes: func(ctx context.Context, payload []byte) ([]byte, error) {
			response, err := service.Projects.ServiceAccounts.SignBlob(
				"projects/-/serviceAccounts/"+mangaImageSignerServiceAccount,
				&iamcredentials.SignBlobRequest{Payload: base64.StdEncoding.EncodeToString(payload)},
			).Context(ctx).Do()
			if err != nil {
				return nil, err
			}
			return base64.StdEncoding.DecodeString(response.SignedBlob)
		},
	}, nil
}

func (s gcsMediaSigner) Issue(ctx context.Context, key string, now time.Time) (string, error) {
	url, err := storage.SignedURL(mangaImageBucket, key, &storage.SignedURLOptions{
		Scheme:         storage.SigningSchemeV4,
		Method:         http.MethodGet,
		Expires:        now.Add(mediaURLTTL),
		GoogleAccessID: mangaImageSignerServiceAccount,
		SignBytes: func(payload []byte) ([]byte, error) {
			return s.signBytes(ctx, payload)
		},
	})
	if err != nil {
		return "", fmt.Errorf("sign GCS URL for %q: %w", key, err)
	}
	return url, nil
}

type manga struct {
	ID     string
	Title  string
	Author string
}

type volume struct {
	ID            string
	Number        int
	Title         string
	PageCount     int
	PageExtension string
}

type pageResponse struct {
	Number   int    `json:"number"`
	ImageURL string `json:"imageUrl"`
}

type app struct {
	verifier       tokenVerifier
	allowed        map[string]struct{}
	gcsMediaSigner gcsMediaSigner
	db             *sql.DB
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
	item, selectedVolume, found, err := a.findVolume(r.Context(), r.PathValue("mangaID"), r.PathValue("volumeID"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not read catalog"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "volume not found"})
		return
	}
	pages, err := a.issuePageBatch(r.Context(), item, selectedVolume, 1)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not issue image URL"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"mangaTitle":   item.Title,
		"volumeNumber": selectedVolume.Number,
		"volumeTitle":  selectedVolume.Title,
		"pageCount":    selectedVolume.PageCount,
		"pages":        pages,
	})
}

func (a app) handleVolumePages(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.authenticate(w, r); !ok {
		return
	}
	start, err := strconv.Atoi(r.URL.Query().Get("start"))
	if err != nil || start < 1 || (start-1)%pageURLBatchSize != 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "start must be a positive page URL batch boundary"})
		return
	}
	item, selectedVolume, found, err := a.findVolume(r.Context(), r.PathValue("mangaID"), r.PathValue("volumeID"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not read catalog"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "volume not found"})
		return
	}
	if start > selectedVolume.PageCount {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "start is beyond the last page"})
		return
	}
	pages, err := a.issuePageBatch(r.Context(), item, selectedVolume, start)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not issue image URL"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"pages": pages})
}

func (a app) findVolume(ctx context.Context, mangaID, volumeID string) (manga, volume, bool, error) {
	var item manga
	var selectedVolume volume
	err := a.db.QueryRowContext(ctx, `
SELECT m.id, m.title, m.author_name, v.id, v.number, v.title, v.page_count, v.page_extension
FROM mangas m JOIN volumes v ON v.manga_id = m.id
WHERE m.id = ? AND v.id = ?`, mangaID, volumeID).Scan(
		&item.ID, &item.Title, &item.Author, &selectedVolume.ID, &selectedVolume.Number, &selectedVolume.Title, &selectedVolume.PageCount, &selectedVolume.PageExtension)
	if errors.Is(err, sql.ErrNoRows) {
		return manga{}, volume{}, false, nil
	}
	if err != nil {
		return manga{}, volume{}, false, err
	}
	return item, selectedVolume, true, nil
}

func (a app) issuePageBatch(ctx context.Context, item manga, selectedVolume volume, start int) ([]pageResponse, error) {
	end := min(start+pageURLBatchSize-1, selectedVolume.PageCount)
	pages := make([]pageResponse, end-start+1)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	var firstErr error
	var errOnce sync.Once
	for number := start; number <= end; number++ {
		index := number - start
		wg.Add(1)
		go func(number, index int) {
			defer wg.Done()
			key := fmt.Sprintf("manga/%s/%s/%03d.%s", item.ID, selectedVolume.ID, number, selectedVolume.PageExtension)
			imageURL, err := a.gcsMediaSigner.Issue(ctx, key, time.Now())
			if err != nil {
				errOnce.Do(func() {
					firstErr = err
					cancel()
				})
				return
			}
			pages[index] = pageResponse{Number: number, ImageURL: imageURL}
		}(number, index)
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return pages, nil
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
	file, err := os.CreateTemp("", "manga-catalog-*.db")
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
	allowedEmails := os.Getenv("ALLOWED_EMAILS")
	if projectID == "" || allowedEmails == "" {
		log.Fatal("FIREBASE_PROJECT_ID and ALLOWED_EMAILS must be set")
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

	gcsSigner, err := newGCSMediaSigner(ctx)
	if err != nil {
		log.Fatalf("initialize GCS media signer: %v", err)
	}

	a := app{verifier: firebaseVerifier{client: client}, allowed: allowed, gcsMediaSigner: gcsSigner, db: catalogDB}
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
	mux.HandleFunc("GET /api/manga/{mangaID}/volumes/{volumeID}/pages", a.handleVolumePages)
	mux.HandleFunc("GET /", servePage("index.html"))
	mux.HandleFunc("GET /library", servePage("library.html"))
	mux.HandleFunc("GET /manga/{mangaID}", servePage("manga.html"))
	mux.HandleFunc("GET /manga/{mangaID}/volumes/{volumeID}", servePage("reader.html"))
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("public/assets"))))
	return mux
}
