package converter

import (
	"fmt"

	"github.com/axsh/tokotachi-scaffolds/features/templatizer/internal/catalog"
)

// ConvertParams holds the parameters for the template conversion pipeline.
type ConvertParams struct {
	// OldModule is the original module path in the source code (e.g. "function").
	OldModule string
	// NewModule is the new module path to replace with (e.g. "github.com/new-org/app").
	NewModule string
	// OldProgram is the original program name (e.g. "function").
	OldProgram string
	// NewProgram is the new program name (e.g. "my-app").
	NewProgram string
	// HintParams provides values for {{param}} placeholders in .hints files.
	HintParams map[string]string
}

// Convert executes the full template conversion pipeline on tempDir.
//
// Processing order:
//
//	Step 1: Cleanup (remove .git, go.sum, vendor, bin, .DS_Store)
//	Step 2: AST transformation (go.mod, *.go → .tmpl postfix)
//	Step 3: Directory rename (cmd/<old> → cmd/<new>)
//	Step 4: Hint file processing (*.hints → apply replacements, .tmpl postfix)
//
// If the params indicate no conversion is needed (empty OldModule), the
// pipeline is skipped entirely.
func Convert(tempDir string, params ConvertParams) error {
	// Skip if no conversion params provided.
	if params.OldModule == "" {
		return nil
	}

	// Step 1: Cleanup
	if err := Clean(tempDir, DefaultExcludes); err != nil {
		return fmt.Errorf("step 1 (cleanup) failed: %w", err)
	}

	// Step 2: AST transformation
	if _, err := TransformGoFiles(tempDir, params.OldModule, params.NewModule); err != nil {
		return fmt.Errorf("step 2 (AST transform) failed: %w", err)
	}

	// Step 3: Directory rename
	if err := RenameDirectories(tempDir, params.OldProgram, params.NewProgram); err != nil {
		return fmt.Errorf("step 3 (rename) failed: %w", err)
	}

	// Step 4: Hint file processing
	if err := ProcessHints(tempDir, params.HintParams); err != nil {
		return fmt.Errorf("step 4 (hints) failed: %w", err)
	}

	return nil
}

// BuildConvertParams constructs ConvertParams from catalog TemplateParams.
// It extracts module_path and program_name parameters from the template params.
// NewModule is set to template variable format (e.g. "{{module_path}}/{{program_name}}")
// so that go.mod module line becomes a template placeholder.
func BuildConvertParams(templateParams []catalog.TemplateParam) ConvertParams {
	if len(templateParams) == 0 {
		return ConvertParams{}
	}

	params := ConvertParams{
		HintParams: make(map[string]string),
	}

	for _, tp := range templateParams {
		// Resolve old_value: explicit old_value takes priority, fallback to default.
		oldValue := tp.OldValue
		if oldValue == "" {
			oldValue = tp.Default
		}

		switch tp.Name {
		case "module_path":
			params.OldModule = oldValue
		case "program_name":
			params.OldProgram = oldValue
			params.NewProgram = oldValue
		}
		// Populate hint params with resolved old_value.
		params.HintParams[tp.Name] = oldValue
	}

	// Construct template variable for module path in go.mod.
	if _, hasModulePath := params.HintParams["module_path"]; hasModulePath {
		if _, hasProgramName := params.HintParams["program_name"]; hasProgramName {
			params.NewModule = "{{module_path}}/{{program_name}}"
		} else {
			params.NewModule = "{{module_path}}"
		}
	}

	return params
}
