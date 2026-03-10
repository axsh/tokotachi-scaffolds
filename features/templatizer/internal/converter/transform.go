package converter

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// TransformResult holds the result of a single file transformation.
type TransformResult struct {
	Path        string // original file path (relative to tempDir)
	Transformed bool   // whether the file was transformed
}

// TransformGoFiles walks tempDir and applies AST transformations to all
// Go source files (*.go) and go.mod. Files that are transformed get a
// .tmpl postfix appended to their name.
func TransformGoFiles(tempDir string, oldModule, newModule string) ([]TransformResult, error) {
	var results []TransformResult

	err := filepath.WalkDir(tempDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		name := d.Name()
		relPath, _ := filepath.Rel(tempDir, path)

		// Process go.mod files.
		if name == "go.mod" {
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return fmt.Errorf("failed to read %s: %w", relPath, readErr)
			}

			transformed, changed, transformErr := TransformGoMod(content, oldModule, newModule)
			if transformErr != nil {
				return fmt.Errorf("failed to transform %s: %w", relPath, transformErr)
			}

			if changed {
				if writeErr := os.WriteFile(path, transformed, 0o644); writeErr != nil {
					return fmt.Errorf("failed to write %s: %w", relPath, writeErr)
				}
				// Rename to .tmpl
				tmplPath := path + ".tmpl"
				if renameErr := os.Rename(path, tmplPath); renameErr != nil {
					return fmt.Errorf("failed to rename %s to .tmpl: %w", relPath, renameErr)
				}
			}

			results = append(results, TransformResult{Path: relPath, Transformed: changed})
			return nil
		}

		// Process *.go files.
		if strings.HasSuffix(name, ".go") {
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return fmt.Errorf("failed to read %s: %w", relPath, readErr)
			}

			transformed, changed, transformErr := TransformGoSource(content, oldModule, newModule)
			if transformErr != nil {
				return fmt.Errorf("failed to transform %s: %w", relPath, transformErr)
			}

			if changed {
				if writeErr := os.WriteFile(path, transformed, 0o644); writeErr != nil {
					return fmt.Errorf("failed to write %s: %w", relPath, writeErr)
				}
				// Rename to .tmpl
				tmplPath := path + ".tmpl"
				if renameErr := os.Rename(path, tmplPath); renameErr != nil {
					return fmt.Errorf("failed to rename %s to .tmpl: %w", relPath, renameErr)
				}
			}

			results = append(results, TransformResult{Path: relPath, Transformed: changed})
		}

		return nil
	})

	return results, err
}
