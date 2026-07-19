package main

import (
	"context"
	"flag"
	"log"

	"github.com/tomozo6/comic/application/internal/catalog"
)

func main() {
	source := flag.String("source", "catalog/mangas", "directory containing manga YAML files")
	output := flag.String("output", "catalog.db", "generated SQLite catalog path")
	flag.Parse()
	if err := catalog.Build(context.Background(), *source, *output); err != nil {
		log.Fatal(err)
	}
}
