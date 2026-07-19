package main

import (
	"net/http"
	"net/http/httptest"
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
