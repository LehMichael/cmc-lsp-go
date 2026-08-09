package lsp

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/lehmichael/cmc-lsp-go/internal/workspace"
)

func TestIncludeDefinitionAndDocumentLink(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "main.upscr")
	targetPath := filepath.Join(root, "Library", "common.uplib")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetPath, []byte("func Common() {\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	const source = "#include \"Library\\common.uplib\"\n#include \"missing.uplib\"\n#include \"$(Up.library)\"\n"
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	uri := workspace.PathToURI(sourcePath)
	targetURI := workspace.PathToURI(targetPath)
	var output bytes.Buffer
	server := NewLsp(bytes.NewReader(nil), &output)
	if err := server.overlay.Open(uri, source, 1); err != nil {
		t.Fatal(err)
	}

	server.handleDefinition(requestMessage{ID: intMessageID(1), Params: marshalParams(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 0, "character": 18},
	})})
	var definitionResponse struct {
		Result *location `json:"result"`
	}
	readResponse(t, &output, &definitionResponse)
	if definitionResponse.Result == nil || definitionResponse.Result.URI != targetURI || definitionResponse.Result.Range != (lspRange{}) {
		t.Fatalf("include definition = %#v", definitionResponse.Result)
	}

	server.handleDocumentLinks(requestMessage{ID: intMessageID(2), Params: marshalParams(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
	})})
	var linksResponse struct {
		Result []documentLink `json:"result"`
	}
	readResponse(t, &output, &linksResponse)
	if len(linksResponse.Result) != 1 || linksResponse.Result[0].Target != targetURI {
		t.Fatalf("document links = %#v", linksResponse.Result)
	}
	if linksResponse.Result[0].Range.Start.Line != 0 || linksResponse.Result[0].Range.Start.Character != 9 {
		t.Fatalf("document link range = %#v", linksResponse.Result[0].Range)
	}
}

func TestMissingIncludeHasNoDefinition(t *testing.T) {
	const uri = "file:///tmp/missing-include.upscr"
	const source = "#include \"missing.uplib\"\n"
	server := NewLsp(bytes.NewReader(nil), &bytes.Buffer{})
	if err := server.overlay.Open(uri, source, 1); err != nil {
		t.Fatal(err)
	}
	if got := server.includeLocationAt(source, uri, position{Line: 0, Character: 15}); got != nil {
		t.Fatalf("missing include location = %#v", got)
	}
}
