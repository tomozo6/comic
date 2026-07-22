package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/tomozo6/manga/application/internal/catalog"
)

type testVerifier struct{}

func (testVerifier) Verify(_ context.Context, _ string) (identity, error) {
	return identity{UID: "test-user", Email: "family@example.com"}, nil
}

func newTestApp(t *testing.T, db *sql.DB) app {
	t.Helper()
	signer, err := newGCSMediaSigner(context.Background())
	if err != nil {
		t.Fatalf("newGCSMediaSigner() error = %v", err)
	}
	return app{verifier: testVerifier{}, allowed: map[string]struct{}{"family@example.com": {}}, gcsMediaSigner: signer, db: db}
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
	a := newTestApp(t, db)
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
	if len(works) != 1 || works[0].ID != "historie" {
		t.Fatalf("works = %#v, want historie", works)
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
	a := newTestApp(t, db)
	req := httptest.NewRequest(http.MethodGet, "/api/manga/historie/volumes/001", nil)
	req.SetPathValue("mangaID", "historie")
	req.SetPathValue("volumeID", "001")
	req.Header.Set("Authorization", "Bearer test-token")
	res := httptest.NewRecorder()

	a.handleVolume(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var body struct {
		PageCount int `json:"pageCount"`
		Pages     []struct {
			Number int    `json:"number"`
			Image  string `json:"imageUrl"`
		} `json:"pages"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.PageCount != 215 || len(body.Pages) != pageURLBatchSize || body.Pages[0].Number != 1 || body.Pages[7].Number != 8 {
		t.Fatalf("pages = %#v, want first batch of historie", body.Pages)
	}
}

func TestVolumePageBatchesUseEightPagesAndKeepOrder(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "catalog.db")
	if err := catalog.Build(context.Background(), "catalog/mangas", dbPath); err != nil {
		t.Fatalf("build catalog: %v", err)
	}
	db, err := catalog.OpenReadonly(dbPath)
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	defer db.Close()
	a := newTestApp(t, db)

	initial := httptest.NewRequest(http.MethodGet, "/api/manga/historie/volumes/001", nil)
	initial.SetPathValue("mangaID", "historie")
	initial.SetPathValue("volumeID", "001")
	initial.Header.Set("Authorization", "Bearer test-token")
	initialResponse := httptest.NewRecorder()
	a.handleVolume(initialResponse, initial)
	if initialResponse.Code != http.StatusOK {
		t.Fatalf("initial status = %d, want %d", initialResponse.Code, http.StatusOK)
	}
	var initialBody struct {
		PageCount int            `json:"pageCount"`
		Pages     []pageResponse `json:"pages"`
	}
	if err := json.NewDecoder(initialResponse.Body).Decode(&initialBody); err != nil {
		t.Fatalf("decode initial response: %v", err)
	}
	if initialBody.PageCount != 215 || len(initialBody.Pages) != pageURLBatchSize || initialBody.Pages[0].Number != 1 || initialBody.Pages[7].Number != 8 {
		t.Fatalf("initial batch = %#v, want pages 1 through 8 of 215", initialBody)
	}

	for _, test := range []struct {
		start      string
		wantStatus int
		wantFirst  int
		wantLast   int
		wantLength int
	}{
		{start: "9", wantStatus: http.StatusOK, wantFirst: 9, wantLast: 16, wantLength: 8},
		{start: "209", wantStatus: http.StatusOK, wantFirst: 209, wantLast: 215, wantLength: 7},
		{start: "2", wantStatus: http.StatusBadRequest},
		{start: "217", wantStatus: http.StatusBadRequest},
	} {
		t.Run("start="+test.start, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/manga/historie/volumes/001/pages?start="+test.start, nil)
			req.SetPathValue("mangaID", "historie")
			req.SetPathValue("volumeID", "001")
			req.Header.Set("Authorization", "Bearer test-token")
			response := httptest.NewRecorder()
			a.handleVolumePages(response, req)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if test.wantStatus != http.StatusOK {
				return
			}
			var body struct {
				Pages []pageResponse `json:"pages"`
			}
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode batch response: %v", err)
			}
			if len(body.Pages) != test.wantLength || body.Pages[0].Number != test.wantFirst || body.Pages[len(body.Pages)-1].Number != test.wantLast {
				t.Fatalf("pages = %#v, want %d through %d", body.Pages, test.wantFirst, test.wantLast)
			}
		})
	}
}
