package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/tomozo6/manga/application/internal/catalog"
)

type testVerifier struct{}

func (testVerifier) Verify(_ context.Context, _ string) (identity, error) {
	return identity{UID: "test-user", Email: "family@example.com"}, nil
}

func TestMangaListUsesFrontendJSONKeys(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "catalog.db")
	if err := catalog.Build(context.Background(), "catalog/mangas", dbPath); err != nil {
		t.Fatalf("build catalog: %v", err)
	}
	db, err := catalog.OpenReadonly(dbPath)
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	defer db.Close()
	a := app{
		verifier: testVerifier{},
		allowed:  map[string]struct{}{"family@example.com": {}},
		signer:   newLocalMediaSigner([]byte("test-secret"), time.Minute),
		db:       db,
	}
	req := httptest.NewRequest(http.MethodGet, "/api/manga", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	res := httptest.NewRecorder()

	a.handleManga(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var works []struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(res.Body).Decode(&works); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(works) == 0 || works[0].ID != "demo-manga" {
		t.Fatalf("works = %#v, want first id demo-manga", works)
	}
}

func TestVolumePagesAreGeneratedFromCatalogMetadata(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "catalog.db")
	if err := catalog.Build(context.Background(), "catalog/mangas", dbPath); err != nil {
		t.Fatalf("build catalog: %v", err)
	}
	db, err := catalog.OpenReadonly(dbPath)
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	defer db.Close()
	a := app{verifier: testVerifier{}, allowed: map[string]struct{}{"family@example.com": {}}, signer: newLocalMediaSigner([]byte("test-secret"), time.Minute), db: db}
	req := httptest.NewRequest(http.MethodGet, "/api/manga/demo-manga/volumes/volume-1", nil)
	req.SetPathValue("mangaID", "demo-manga")
	req.SetPathValue("volumeID", "volume-1")
	req.Header.Set("Authorization", "Bearer test-token")
	res := httptest.NewRecorder()

	a.handleVolume(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var body struct {
		Pages []struct {
			Number int    `json:"number"`
			Image  string `json:"imageUrl"`
		} `json:"pages"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Pages) != 3 || body.Pages[2].Number != 3 {
		t.Fatalf("pages = %#v, want three ordered pages", body.Pages)
	}
}
