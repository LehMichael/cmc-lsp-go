package analysis

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lehmichael/cmc-lsp-go/internal/diag"
	"github.com/lehmichael/cmc-lsp-go/internal/workspace"
)

func TestAnalyzeIncludesAndLibraryDefinitions(t *testing.T) {
	directory := t.TempDir()
	root := filepath.Join(directory, "main.upscr")
	library := filepath.Join(directory, "lib.uplib")
	if err := os.WriteFile(root, []byte("#include \"lib.uplib\"\nUp.value = Helper(1)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(library, []byte("func Helper($) {\n  Return(Up.$1)\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	symbols, diagnostics := Analyze(root, workspace.NewOverlay())
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if len(symbols) < 3 {
		t.Fatalf("got %d symbols, want at least function, assignment, and call", len(symbols))
	}
}

func TestAnalyzeMissingAndCircularIncludes(t *testing.T) {
	directory := t.TempDir()
	root := filepath.Join(directory, "main.upscr")
	other := filepath.Join(directory, "other.upscr")
	if err := os.WriteFile(root, []byte("#include \"other.upscr\"\n#include \"missing.upscr\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("#include \"main.upscr\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, diagnostics := Analyze(root, workspace.NewOverlay())
	seen := map[diag.DiagnosticKind]bool{}
	for _, item := range diagnostics {
		seen[item.Kind] = true
	}
	if !seen[diag.MissingInclude] || !seen[diag.CircularInclude] {
		t.Fatalf("expected missing and circular include diagnostics, got %#v", diagnostics)
	}
}
