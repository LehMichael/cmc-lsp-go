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
	Ast   parser.Ast
	Token []lexer.Token
	Diag  []diag.Diagnostic
}

func Analyze(root string, ws *workspace.Overlay) ([]Symbol, []diag.Diagnostic) {
	parsed := resolve(root, ws)
	var di []diag.Diagnostic
	var sym []Symbol
	seen := make(map[string]*parser.SectionSwitchKind)

	var walk func(string, *parser.SectionSwitchKind, map[string]struct{}) *parser.SectionSwitchKind
	walk = func(u string, ssw *parser.SectionSwitchKind, stack map[string]struct{}) *parser.SectionSwitchKind {
		stack[u] = struct{}{}

		p := parsed[u]

		di = append(di, p.Diag...)

		basePath := filepath.Dir(u)
		fileType := strings.ToLower(filepath.Ext(u))

		for _, s := range p.Ast {
			if pp, ok := s.Kind.(parser.PreprocessorStatement); ok {
				if inc, ok := pp.Kind.(parser.IncludePpStatement); ok {
					// remove quotes
					includePath := strings.Trim(inc.Path, "\"")
					path := includeFilePath(basePath, includePath)
					// is file is already on the stack, append diag and continue
					if _, ok := stack[path]; ok {
						di = append(di, diag.Diagnostic{
							Kind:     diag.CircularInclude,
							Range:    s.Range,
							Severity: diag.Error,
						})
						continue
					}

					// already parsed this file, just push the section
					if s, ok := seen[path]; ok {
						ssw = s
						continue
					}

					// file does not exist, diag was already appended
					if _, ok := parsed[path]; !ok {
						continue
					}

					ssw = walk(path, ssw, maps.Clone(stack))
					seen[path] = ssw
				}
			} else if sec, ok := s.Kind.(parser.SectionSwitch); ok {
				ssw = &sec.Kind
			} else if as, ok := s.Kind.(parser.Assignment); ok {
				sym = append(sym, Symbol{
					Identifier: as.Target,
					Section:    ssw,
				})
				sym = append(sym, resolveExpression(as.Value.Kind, ssw)...)
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
				if fileType != ".uplib" {
					di = append(di, diag.Diagnostic{
						Kind:     diag.FunctionInScript,
						Range:    s.Range,
						Severity: diag.Error,
					})
				}
			} else if cs, ok := s.Kind.(parser.CallStatement); ok {
				sym = append(sym, resolveExpression(parser.CallExpression(cs), ssw)...)
			}
		}

		return ssw
	}

	walk(root, nil, make(map[string]struct{}))

	return sym, di
}

func resolveExpression(ex parser.ExpressionKind, ssw *parser.SectionSwitchKind) []Symbol {
	if ge, ok := ex.(parser.GroupedExpression); ok {
		return resolveExpression(ge.Expression.Kind, ssw)
	} else if pe, ok := ex.(parser.PrefixedExpression); ok {
		return resolveExpression(pe.Expression.Kind, ssw)
	} else if be, ok := ex.(parser.BinaryExpression); ok {
		ret := resolveExpression(be.Left.Kind, ssw)
		ret = append(ret, resolveExpression(be.Right.Kind, ssw)...)
		return ret
	} else if ie, ok := ex.(parser.IdentifierExpression); ok {
		return []Symbol{{
			Identifier: ie,
			Section:    ssw,
		}}
	} else if ce, ok := ex.(parser.CallExpression); ok {
		ret := []Symbol{{
			Identifier: ce.Identifier,
			Section:    ssw,
		}}
		for _, arg := range ce.Parameters {
			ret = append(ret, resolveExpression(arg.Kind, ssw)...)
		}
		return ret
	}

	return []Symbol{}
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
					path := includeFilePath(basePath, includePath)
					if ok := visit(path); !ok {
						di = append(di, diag.Diagnostic{
							Kind:     diag.MissingInclude,
							Range:    s.Range,
							Severity: diag.Error,
						})
					}
				}
			}
		}

		results[u] = Parsed{ast, tokens, di}

		return true
	}
	visit(root)
	return results
}

func includeFilePath(basePath, includePath string) string {
	// CMC projects are authored on Windows, so relative includes commonly use
	// backslashes even when the language server runs on another platform.
	includePath = filepath.FromSlash(strings.ReplaceAll(includePath, "\\", "/"))
	return filepath.Clean(filepath.Join(basePath, includePath))
}
