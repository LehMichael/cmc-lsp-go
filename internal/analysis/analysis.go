package analysis

import (
	"maps"
	"path/filepath"
	"strings"

	"github.com/lehmichael/cmc-lsp-go/internal/diag"
	"github.com/lehmichael/cmc-lsp-go/internal/lexer"
	"github.com/lehmichael/cmc-lsp-go/internal/parser"
	"github.com/lehmichael/cmc-lsp-go/internal/workspace"
)

type Symbol struct {
	Section    *parser.SectionSwitchKind
	Identifier parser.IdentifierExpression
}

type Parsed struct {
	Ast  parser.Ast
	Diag []diag.Diagnostic
}

func Analyze(root string, ws *workspace.Overlay) ([]Symbol, []diag.Diagnostic) {
	parsed := resolve(root, ws)
	var di []diag.Diagnostic
	var sym []Symbol
	seen := make(map[string]parser.SectionSwitchKind)

	var walk func(string, *parser.SectionSwitchKind, map[string]struct{}) *parser.SectionSwitchKind
	walk = func(u string, ssw *parser.SectionSwitchKind, stack map[string]struct{}) *parser.SectionSwitchKind {
		stack[u] = struct{}{}

		p := parsed[u]

		di = append(di, p.Diag...)

		basePath := filepath.Dir(u)

		for _, s := range p.Ast {
			if pp, ok := s.Kind.(parser.PreprocessorStatement); ok {
				if inc, ok := pp.Kind.(parser.IncludePpStatement); ok {
					// remove quotes
					includePath := strings.Trim(inc.Path, "\"")
					path := filepath.Join(basePath, includePath)
					// is file is already on the stack, addpend diag and continue
					if _, ok := stack[path]; ok {
						di = append(di, diag.Diagnostic{
							Kind:  diag.CircularInclude,
							Range: s.Range,
						})
						continue
					}

					// already parsed this file, just push the section
					if s, ok := seen[path]; ok {
						ssw = &s
						continue
					}

					// file does not exist, diag was already appended
					if _, ok := parsed[path]; !ok {
						continue
					}

					ssw = walk(path, ssw, maps.Clone(stack))
					seen[path] = *ssw
				}
			} else if sec, ok := s.Kind.(parser.SectionSwitch); ok {
				ssw = &sec.Kind
			} else if as, ok := s.Kind.(parser.Assignment); ok {
				sym = append(sym, Symbol{
					Identifier: as.Target,
					Section:    ssw,
				})
				if i, ok := as.Value.Kind.(parser.IdentifierExpression); ok {
					sym = append(sym, Symbol{
						Identifier: i,
						Section:    ssw,
					})
				}
			} else if ds, ok := s.Kind.(parser.DeleteStatement); ok {
				sym = append(sym, Symbol{
					Identifier: ds.Identifier,
					Section:    ssw,
				})
			} else if fs, ok := s.Kind.(parser.FunctionStatement); ok {
				if fs.Identifier == nil {
					continue
				}
				sym = append(sym, Symbol{
					Identifier: *fs.Identifier,
					Section:    ssw,
				})
			} else if cs, ok := s.Kind.(parser.CallStatement); ok {
				sym = append(sym, Symbol{
					Identifier: cs.Identifier,
					Section:    ssw,
				})
				for _, arg := range cs.Parameters {
					if i, ok := arg.Kind.(parser.IdentifierExpression); ok {
						sym = append(sym, Symbol{
							Identifier: i,
							Section:    ssw,
						})
					}
				}
			}
		}

		return ssw
	}

	walk(root, nil, make(map[string]struct{}))

	return sym, di
}

func resolve(root string, ws *workspace.Overlay) map[string]Parsed {
	seen := make(map[string]struct{})
	results := make(map[string]Parsed)

	var visit func(string) bool
	visit = func(u string) bool {
		if _, ok := seen[u]; ok {
			return true // already parsed this run
		}
		seen[u] = struct{}{}

		t, err := ws.Read(u)
		if err != nil {
			return false
		}

		tokens, di := lexer.Tokenize(t)
		ast, di := parser.Parse(tokens, di)

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

		results[u] = Parsed{ast, di}

		return true
	}
	visit(root)
	return results
}
