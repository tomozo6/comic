package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLocalMediaSigner(t *testing.T) {
	signer := newLocalMediaSigner([]byte("test-secret"), time.Minute)
	token, err := signer.Sign("demo/first-01.svg", time.Now())
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	key, err := signer.Verify(token, time.Now().Add(30*time.Second))
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if key != "demo/first-01.svg" {
		t.Errorf("Verify() key = %q, want demo/first-01.svg", key)
	}
}

func TestGCSMediaSignerIssuesOneHourV4URL(t *testing.T) {
	now := time.Now().UTC()
	signer := gcsMediaSigner{
		bucket:         "tomozo-manga-images",
		googleAccessID: "manga-media-signer@tomozo6.iam.gserviceaccount.com",
		ttl:            time.Hour,
		signBytes: func(_ context.Context, payload []byte) ([]byte, error) {
			if len(payload) == 0 {
				t.Fatal("signing payload is empty")
			}
			return []byte("test-signature"), nil
		},
	}

	rawURL, err := signer.Issue(context.Background(), "manga/historie/001/001.jpeg", now)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	issued, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse signed URL: %v", err)
	}
	if issued.Query().Get("X-Goog-Algorithm") != "GOOG4-RSA-SHA256" {
		t.Errorf("algorithm = %q, want V4", issued.Query().Get("X-Goog-Algorithm"))
	}
	expires, err := strconv.Atoi(issued.Query().Get("X-Goog-Expires"))
	if err != nil || expires < 3599 || expires > 3600 {
		t.Errorf("expires = %q, want approximately 3600", issued.Query().Get("X-Goog-Expires"))
	}
	if issued.Query().Get("X-Goog-Credential") == "" {
		t.Error("signed URL has no Google access ID")
	}
}

func TestGCSMediaSignerIntegration(t *testing.T) {
	if os.Getenv("RUN_GCS_INTEGRATION") != "1" {
		t.Skip("set RUN_GCS_INTEGRATION=1 to verify local ADC against GCS")
	}
	signer, err := newGCSMediaSigner(
		context.Background(),
		"tomozo-manga-images",
		"manga-media-signer@tomozo6.iam.gserviceaccount.com",
		time.Hour,
	)
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

func TestLocalMediaSignerRejectsExpiredToken(t *testing.T) {
	signer := newLocalMediaSigner([]byte("test-secret"), time.Minute)
	token, err := signer.Sign("demo/first-01.svg", time.Now().Add(-2*time.Minute))
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	if _, err := signer.Verify(token, time.Now()); err == nil {
		t.Fatal("Verify() accepted an expired token")
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
