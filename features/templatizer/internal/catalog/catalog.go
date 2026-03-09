package catalog

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Scaffold represents a single scaffold entry from catalog.yaml.
type Scaffold struct {
	Name        string `yaml:"name"`
	Category    string `yaml:"category"`
	Description string `yaml:"description"`
	TemplateRef string `yaml:"template_ref"`
	OriginalRef string `yaml:"original_ref"`
}

// Catalog represents the top-level catalog.yaml structure.
type Catalog struct {
	Version   string     `yaml:"version"`
	Scaffolds []Scaffold `yaml:"scaffolds"`
}

// ParseCatalog parses the YAML bytes and returns a Catalog.
// Returns error if the YAML is invalid.
func ParseCatalog(data []byte) (*Catalog, error) {
	var cat Catalog
	if err := yaml.Unmarshal(data, &cat); err != nil {
		return nil, fmt.Errorf("failed to parse catalog YAML: %w", err)
	}
	return &cat, nil
}

// LoadCatalog reads a catalog.yaml file and returns a Catalog.
// Returns error if the file cannot be read or parsed.
func LoadCatalog(path string) (*Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read catalog file %s: %w", path, err)
	}
	return ParseCatalog(data)
}
