package catalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCatalog(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantLen   int
		wantFirst *Scaffold
		wantErr   bool
	}{
		{
			name: "valid catalog with 3 scaffolds",
			input: `version: "1.0.0"
scaffolds:
  - name: "default"
    category: "root"
    description: "Tokotachi - The First of All"
    template_ref: "catalog/templates/root/project-default"
    original_ref: "catalog/originals/root/project-default"
  - name: "axsh-go-standard"
    category: "project"
    description: "AXSH Go Standard Project"
    template_ref: "catalog/templates/axsh/go-standard-project"
    original_ref: "catalog/originals/axsh/go-standard-project"
  - name: "axsh-go-standard"
    category: "feature"
    description: "AXSH Go Standard Feature"
    template_ref: "catalog/templates/axsh/go-standard-feature"
    original_ref: "catalog/originals/axsh/go-standard-feature"
`,
			wantLen: 3,
			wantFirst: &Scaffold{
				Name:        "default",
				Category:    "root",
				Description: "Tokotachi - The First of All",
				TemplateRef: "catalog/templates/root/project-default",
				OriginalRef: "catalog/originals/root/project-default",
			},
			wantErr: false,
		},
		{
			name: "empty scaffolds list",
			input: `version: "1.0.0"
scaffolds: []
`,
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "invalid yaml",
			input:   `{{{invalid yaml`,
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cat, err := ParseCatalog([]byte(tt.input))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, cat.Scaffolds, tt.wantLen)
			if tt.wantFirst != nil && len(cat.Scaffolds) > 0 {
				assert.Equal(t, tt.wantFirst.Name, cat.Scaffolds[0].Name)
				assert.Equal(t, tt.wantFirst.Category, cat.Scaffolds[0].Category)
				assert.Equal(t, tt.wantFirst.TemplateRef, cat.Scaffolds[0].TemplateRef)
				assert.Equal(t, tt.wantFirst.OriginalRef, cat.Scaffolds[0].OriginalRef)
			}
		})
	}
}
