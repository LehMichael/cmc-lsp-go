package analysis

import (
	"path/filepath"
	"strings"

	"github.com/lehmichael/cmc-lsp-go/internal/diag"
	"github.com/lehmichael/cmc-lsp-go/internal/lexer"
	"github.com/lehmichael/cmc-lsp-go/internal/parser"
	"github.com/lehmichael/cmc-lsp-go/internal/source"
	"github.com/lehmichael/cmc-lsp-go/internal/workspace"
)

type Symbol struct {
	Usage      []source.SourceRange
	Section    parser.SectionSwitchKind
	Identifier parser.IdentifierExpression
}

type Parsed struct {
	Ast  parser.Ast
	Diag []diag.Diagnostic
}

func resolve(root string, ws *workspace.Overlay) []Parsed {
	seen := make(map[string]bool)
	var results []Parsed

	var visit func(string) bool
	visit = func(u string) bool {
		if _, ok := seen[u]; ok {
			return true // already parsed this run — and breaks include cycles too
		}
		t, err := ws.Read(u)
		if err != nil {
			return false
		}

		tokens, di := lexer.Tokenize(t)
		ast, di := parser.Parse(tokens, di)

		seen[u] = true

		basePath := filepath.Dir(u)

		for _, s := range ast {
			if pp, ok := s.Kind.(parser.PreprocessorStatement); ok {
				if inc, ok := pp.Kind.(parser.IncludePpStatement); ok {
					// remove quotes
					includePath := strings.Trim(inc.Path, "\"")
					path := filepath.Join(basePath, includePath)
					if ok := visit(path); !ok {
						di = append(di, diag.Diagnostic{
							Kind:  diag.MissingInclude,
							Range: s.Range,
						})
					}
				}
			}
		}

		results = append(results, Parsed{ast, di})

		return true
	}
	visit(root)
	return results
}
