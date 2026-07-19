// Package catalog builds and opens the read-only comic catalog database.
package catalog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)

type mangaFile struct {
	ID             string   `yaml:"id"`
	Title          string   `yaml:"title"`
	AuthorName     string   `yaml:"author_name"`
	CoverObjectKey string   `yaml:"cover_object_key"`
	Volumes        []volume `yaml:"volumes"`
}

type volume struct {
	ID             string `yaml:"id"`
	Number         int    `yaml:"number"`
	Title          string `yaml:"title"`
	CoverObjectKey string `yaml:"cover_object_key"`
	PageCount      int    `yaml:"page_count"`
	PageExtension  string `yaml:"page_extension"`
}

var extension = regexp.MustCompile(`^[A-Za-z0-9]+$`)

// Build validates YAML files in sourceDir and atomically replaces outputPath with a SQLite catalog.
func Build(ctx context.Context, sourceDir, outputPath string) error {
	items, err := load(sourceDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create catalog directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(outputPath), ".catalog-*.db")
	if err != nil {
		return fmt.Errorf("create temporary catalog: %w", err)
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		return err
	}
	defer os.Remove(tmpPath)

	db, err := sql.Open("sqlite", tmpPath)
	if err != nil {
		return fmt.Errorf("open catalog: %w", err)
	}
	if err := write(ctx, db, items); err != nil {
		db.Close()
		return err
	}
	if err := db.Close(); err != nil {
		return fmt.Errorf("close catalog: %w", err)
	}
	if err := os.Rename(tmpPath, outputPath); err != nil {
		return fmt.Errorf("replace catalog: %w", err)
	}
	return nil
}

// OpenReadonly opens a previously generated catalog without permitting writes.
func OpenReadonly(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open catalog read-only: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("open catalog read-only: %w", err)
	}
	return db, nil
}

func load(sourceDir string) ([]mangaFile, error) {
	paths, err := filepath.Glob(filepath.Join(sourceDir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no catalog YAML files found in %s", sourceDir)
	}
	sort.Strings(paths)
	seen := make(map[string]struct{})
	items := make([]mangaFile, 0, len(paths))
	for _, path := range paths {
		contents, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var item mangaFile
		if err := yaml.Unmarshal(contents, &item); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		if err := validate(item); err != nil {
			return nil, fmt.Errorf("validate %s: %w", path, err)
		}
		if _, ok := seen[item.ID]; ok {
			return nil, fmt.Errorf("duplicate manga id %q", item.ID)
		}
		seen[item.ID] = struct{}{}
		items = append(items, item)
	}
	return items, nil
}

func validate(item mangaFile) error {
	if item.ID == "" || item.Title == "" || item.AuthorName == "" {
		return errors.New("id, title, and author_name are required")
	}
	if len(item.Volumes) == 0 {
		return errors.New("at least one volume is required")
	}
	ids := make(map[string]struct{})
	numbers := make(map[int]struct{})
	for _, volume := range item.Volumes {
		if volume.ID == "" || volume.Title == "" || volume.Number < 1 || volume.PageCount < 1 {
			return errors.New("volume id, title, positive number, and positive page_count are required")
		}
		if !extension.MatchString(volume.PageExtension) {
			return fmt.Errorf("volume %q has invalid page_extension %q", volume.ID, volume.PageExtension)
		}
		if _, ok := ids[volume.ID]; ok {
			return fmt.Errorf("duplicate volume id %q", volume.ID)
		}
		if _, ok := numbers[volume.Number]; ok {
			return fmt.Errorf("duplicate volume number %d", volume.Number)
		}
		ids[volume.ID] = struct{}{}
		numbers[volume.Number] = struct{}{}
	}
	return nil
}

func write(ctx context.Context, db *sql.DB, items []mangaFile) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
PRAGMA foreign_keys = ON;
CREATE TABLE mangas (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  author_name TEXT NOT NULL,
  cover_object_key TEXT NOT NULL DEFAULT ''
);
CREATE TABLE volumes (
  manga_id TEXT NOT NULL REFERENCES mangas(id),
  id TEXT NOT NULL,
  number INTEGER NOT NULL CHECK (number > 0),
  title TEXT NOT NULL,
  cover_object_key TEXT NOT NULL DEFAULT '',
  page_count INTEGER NOT NULL CHECK (page_count > 0),
  page_extension TEXT NOT NULL,
  PRIMARY KEY (manga_id, id),
  UNIQUE (manga_id, number)
);`); err != nil {
		return fmt.Errorf("create catalog schema: %w", err)
	}
	for _, item := range items {
		if _, err := tx.ExecContext(ctx, `INSERT INTO mangas (id, title, author_name, cover_object_key) VALUES (?, ?, ?, ?)`, item.ID, item.Title, item.AuthorName, item.CoverObjectKey); err != nil {
			return err
		}
		for _, volume := range item.Volumes {
			if _, err := tx.ExecContext(ctx, `INSERT INTO volumes (manga_id, id, number, title, cover_object_key, page_count, page_extension) VALUES (?, ?, ?, ?, ?, ?, ?)`, item.ID, volume.ID, volume.Number, volume.Title, volume.CoverObjectKey, volume.PageCount, volume.PageExtension); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}
