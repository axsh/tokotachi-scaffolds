package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/axsh/tokotachi-scaffolds/features/templatizer/internal/archiver"
	"github.com/axsh/tokotachi-scaffolds/features/templatizer/internal/catalog"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <catalog.yaml>\n", os.Args[0])
		os.Exit(1)
	}

	catalogPath := os.Args[1]

	cat, err := catalog.LoadCatalog(catalogPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Resolve paths relative to the directory containing catalog.yaml
	baseDir := filepath.Dir(catalogPath)

	fmt.Printf("Processing %d scaffold(s)...\n", len(cat.Scaffolds))

	for _, s := range cat.Scaffolds {
		originalDir := filepath.Join(baseDir, s.OriginalRef)
		zipPath := filepath.Join(baseDir, s.TemplateRef+".zip")

		fmt.Printf("  Archiving %s -> %s\n", originalDir, zipPath)

		if err := archiver.ZipDirectory(originalDir, zipPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error archiving scaffold %q: %v\n", s.Name, err)
			os.Exit(1)
		}
	}

	fmt.Println("Done. All scaffolds archived successfully.")
}
