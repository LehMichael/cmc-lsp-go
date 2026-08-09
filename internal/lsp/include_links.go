package lsp

import (
	"path/filepath"
	"strings"

	"github.com/lehmichael/cmc-lsp-go/internal/lexer"
	"github.com/lehmichael/cmc-lsp-go/internal/workspace"
)

type includeTarget struct {
	rangeValue lspRange
	uri        string
}

func (server *Lsp) handleDocumentLinks(message requestMessage) {
	var params documentParams
	if err := decodeParams(message.Params, &params); err != nil {
		server.respond(message.ID, nil, &responseError{Code: InvalidParams, Message: err.Error()})
		return
	}
	text, err := server.overlay.ReadURI(params.TextDocument.URI)
	if err != nil {
		server.respond(message.ID, []documentLink{}, nil)
		return
	}
	targets := server.includeTargets(text, params.TextDocument.URI)
	links := make([]documentLink, 0, len(targets))
	for _, target := range targets {
		links = append(links, documentLink{
			Range: target.rangeValue, Target: target.uri, Tooltip: "Open included CMC file",
		})
	}
	server.respond(message.ID, links, nil)
}

func (server *Lsp) includeLocationAt(text, uri string, target position) *location {
	for _, include := range server.includeTargets(text, uri) {
		if positionInRange(target, include.rangeValue) {
			return &location{URI: include.uri, Range: lspRange{}}
		}
	}
	return nil
}

func (server *Lsp) includeTargets(text, uri string) []includeTarget {
	tokens, _ := lexer.Tokenize(cmcTextForURI(uri, text))
	expectingPath := false
	var result []includeTarget
	for _, token := range tokens {
		switch token.Kind {
		case lexer.PreprocessorInclude:
			expectingPath = true
		case lexer.LiteralString:
			if !expectingPath {
				continue
			}
			expectingPath = false
			if targetURI, ok := server.resolveIncludeURI(uri, token.Lexeme); ok {
				result = append(result, includeTarget{rangeValue: sourceRangeToLSP(text, token.Range), uri: targetURI})
			}
		case lexer.NewLine, lexer.EOF, lexer.Comment:
			expectingPath = false
		default:
			if expectingPath {
				expectingPath = false
			}
		}
	}
	return result
}

func (server *Lsp) resolveIncludeURI(sourceURI, include string) (string, bool) {
	sourcePath, err := workspace.URIToPath(sourceURI)
	if err != nil {
		return "", false
	}
	include = strings.TrimSpace(include)
	if len(include) >= 2 && include[0] == '"' && include[len(include)-1] == '"' {
		include = include[1 : len(include)-1]
	}
	if include == "" || strings.Contains(include, "$(") {
		return "", false
	}
	include = filepath.FromSlash(strings.ReplaceAll(include, "\\", "/"))
	target := include
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(sourcePath), target)
	}
	target = filepath.Clean(target)
	if _, err := server.overlay.Read(target); err != nil {
		return "", false
	}
	return workspace.PathToURI(target), true
}
