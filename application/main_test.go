package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGCSMediaSignerReadsHistorieImage(t *testing.T) {
	signer, err := newGCSMediaSigner(context.Background())
	if err != nil {
		t.Fatalf("newGCSMediaSigner() error = %v", err)
	}
	rawURL, err := signer.Issue(context.Background(), "manga/historie/001/001.jpeg", time.Now())
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	response, err := http.Get(rawURL)
	if err != nil {
		t.Fatalf("GET signed URL: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET signed URL status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		t.Fatalf("read signed URL response: %v", err)
	}
}

func TestRouterServesFrontendAssets(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/assets/js/api.js", nil)
	response := httptest.NewRecorder()

	newRouter(app{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), "fetchAPI") {
		t.Fatal("asset response does not contain expected JavaScript")
	}
}
