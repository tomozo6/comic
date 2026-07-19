package catalog

import (
	"context"
	"path/filepath"
	"testing"
)

func TestBuildCreatesCatalogFromYAML(t *testing.T) {
	output := filepath.Join(t.TempDir(), "catalog.db")
	if err := Build(context.Background(), "../../catalog/mangas", output); err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	db, err := OpenReadonly(output)
	if err != nil {
		t.Fatalf("OpenReadonly() error = %v", err)
	}
	defer db.Close()
	var title string
	if err := db.QueryRow(`SELECT title FROM volumes WHERE manga_id = ? AND id = ?`, "demo-comic", "volume-1").Scan(&title); err != nil {
		t.Fatalf("query volume: %v", err)
	}
	if title != "青い本のひみつ" {
		t.Errorf("title = %q, want %q", title, "青い本のひみつ")
	}
}
