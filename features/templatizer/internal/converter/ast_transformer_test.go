package converter

import (
	"testing"
)

func TestTransformGoMod(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		oldModule   string
		newModule   string
		want        string
		wantChanged bool
	}{
		{
			name:        "replaces simple module name",
			input:       "module function\n\ngo 1.24.0\n",
			oldModule:   "function",
			newModule:   "github.com/new-org/new-app",
			want:        "module github.com/new-org/new-app\n\ngo 1.24.0\n",
			wantChanged: true,
		},
		{
			name:        "replaces full module path",
			input:       "module github.com/old-org/old-app\n\ngo 1.24.0\n",
			oldModule:   "github.com/old-org/old-app",
			newModule:   "github.com/new-org/new-app",
			want:        "module github.com/new-org/new-app\n\ngo 1.24.0\n",
			wantChanged: true,
		},
		{
			name: "preserves require block",
			input: "module function\n\ngo 1.24.0\n\n" +
				"require github.com/axsh/kuniumi v0.1.5\n",
			oldModule: "function",
			newModule: "github.com/new-org/new-app",
			want: "module github.com/new-org/new-app\n\ngo 1.24.0\n\n" +
				"require github.com/axsh/kuniumi v0.1.5\n",
			wantChanged: true,
		},
		{
			name:        "no change when module does not match",
			input:       "module other-module\n\ngo 1.24.0\n",
			oldModule:   "function",
			newModule:   "github.com/new-org/new-app",
			want:        "module other-module\n\ngo 1.24.0\n",
			wantChanged: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed, err := TransformGoMod([]byte(tt.input), tt.oldModule, tt.newModule)
			if err != nil {
				t.Fatalf("TransformGoMod() error: %v", err)
			}
			if changed != tt.wantChanged {
				t.Errorf("TransformGoMod() changed = %v, want %v", changed, tt.wantChanged)
			}
			if string(got) != tt.want {
				t.Errorf("TransformGoMod() output:\ngot:\n%s\nwant:\n%s", string(got), tt.want)
			}
		})
	}
}

func TestTransformGoSource(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		oldModule   string
		newModule   string
		want        string
		wantChanged bool
	}{
		{
			name: "replaces matching import path",
			input: `package main

import "function/internal/function"

func main() {}
`,
			oldModule: "function",
			newModule: "github.com/new-org/new-app",
			want: `package main

import "github.com/new-org/new-app/internal/function"

func main() {}
`,
			wantChanged: true,
		},
		{
			name: "replaces multiple matching imports",
			input: `package main

import (
	"function/internal/config"
	"function/internal/server"
	"fmt"
)

func main() {}
`,
			oldModule: "function",
			newModule: "github.com/new-org/new-app",
			want: `package main

import (
	"fmt"
	"github.com/new-org/new-app/internal/config"
	"github.com/new-org/new-app/internal/server"
)

func main() {}
`,
			wantChanged: true,
		},
		{
			name: "does not replace external packages",
			input: `package main

import (
	"function/internal/function"
	"github.com/spf13/cobra"
	"fmt"
)

func main() {}
`,
			oldModule: "function",
			newModule: "github.com/new-org/new-app",
			want: `package main

import (
	"fmt"
	"github.com/new-org/new-app/internal/function"
	"github.com/spf13/cobra"
)

func main() {}
`,
			wantChanged: true,
		},
		{
			name: "does not change comment containing module path",
			input: `package main

// This uses function/internal/function for something
import "fmt"

func main() {}
`,
			oldModule:   "function",
			newModule:   "github.com/new-org/new-app",
			want:        "", // no change expected, compare with input
			wantChanged: false,
		},
		{
			name: "does not change string literal containing module path",
			input: `package main

import "fmt"

func main() {
	fmt.Println("function/internal/function")
}
`,
			oldModule:   "function",
			newModule:   "github.com/new-org/new-app",
			want:        "", // no change expected
			wantChanged: false,
		},
		{
			name: "replaces exact module import (no subpath)",
			input: `package main

import "function"

func main() {}
`,
			oldModule: "function",
			newModule: "github.com/new-org/new-app",
			want: `package main

import "github.com/new-org/new-app"

func main() {}
`,
			wantChanged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed, err := TransformGoSource([]byte(tt.input), tt.oldModule, tt.newModule)
			if err != nil {
				t.Fatalf("TransformGoSource() error: %v", err)
			}
			if changed != tt.wantChanged {
				t.Errorf("TransformGoSource() changed = %v, want %v", changed, tt.wantChanged)
			}
			if tt.wantChanged {
				if string(got) != tt.want {
					t.Errorf("TransformGoSource() output:\ngot:\n%s\nwant:\n%s", string(got), tt.want)
				}
			}
		})
	}
}
