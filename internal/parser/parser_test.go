package parser

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lehmichael/cmc-lsp-go/internal/lexer"

	"github.com/google/go-cmp/cmp"
)

var update = flag.Bool("update", false, "update golden files")

func TestParser(t *testing.T) {
	matches, _ := filepath.Glob("testdata/*.upscr")
	for _, input := range matches {
		t.Run(filepath.Base(input), func(t *testing.T) {
			src, err := os.ReadFile(input)
			if err != nil {
				t.Fatal(err)
			}

			tokens, diagnostics := lexer.Tokenize(string(src))
			ast, diagnostics := Parse(tokens, diagnostics)

			gotAst, err := json.MarshalIndent(ast, "", "  ")
			if err != nil {
				t.Fatal(err)
			}

			gotDiag, err := json.MarshalIndent(diagnostics, "", "  ")
			if err != nil {
				t.Fatal(err)
			}

			goldenAstPath := strings.TrimSuffix(input, ".upscr") + ".ast.json"
			goldenDiagPath := strings.TrimSuffix(input, ".upscr") + ".diag.json"
			if *update {
				os.WriteFile(goldenAstPath, gotAst, 0o644)
				os.WriteFile(goldenDiagPath, gotDiag, 0o644)
				return
			}

			wantAst, err := os.ReadFile(goldenAstPath)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(string(wantAst), string(gotAst)); diff != "" {
				t.Errorf("AST mismatch (-want +got):\n%s", diff)
			}

			wantDiag, err := os.ReadFile(goldenDiagPath)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(string(wantDiag), string(gotDiag)); diff != "" {
				t.Errorf("Diag mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
