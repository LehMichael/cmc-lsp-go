package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
)

func TestServerDocumentLifecycleAndFormatting(t *testing.T) {
	uri := "file:///tmp/example.upscr"
	messages := []any{
		map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{"capabilities": map[string]any{}}},
		map[string]any{"jsonrpc": "2.0", "method": "initialized", "params": map[string]any{}},
		map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{
			"textDocument": map[string]any{"uri": uri, "languageId": "cmc", "version": 1, "text": "IF(up.x==1)\nLog(\"x\")\nENDIF"},
		}},
		map[string]any{"jsonrpc": "2.0", "id": "format", "method": "textDocument/formatting", "params": map[string]any{
			"textDocument": map[string]any{"uri": uri}, "options": map[string]any{"tabSize": 2, "insertSpaces": true},
		}},
		map[string]any{"jsonrpc": "2.0", "id": "tokens", "method": "textDocument/semanticTokens/full", "params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
		}},
		map[string]any{"jsonrpc": "2.0", "method": "textDocument/didChange", "params": map[string]any{
			"textDocument": map[string]any{"uri": uri, "version": 2}, "contentChanges": []map[string]any{{"text": "If true\n"}},
		}},
		map[string]any{"jsonrpc": "2.0", "id": 3, "method": "shutdown"},
		map[string]any{"jsonrpc": "2.0", "method": "exit"},
	}

	var input bytes.Buffer
	for _, message := range messages {
		payload, err := json.Marshal(message)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&input, "Content-Length: %d\r\n\r\n", len(payload))
		input.Write(payload)
	}
	var output bytes.Buffer
	if exitCode := NewLsp(&input, &output).Start(); exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}

	reader := bufio.NewReader(&output)
	var got []map[string]any
	for reader.Buffered() > 0 || output.Len() > 0 {
		payload, err := readFrame(reader)
		if err != nil {
			break
		}
		var message map[string]any
		if err := json.Unmarshal(payload, &message); err != nil {
			t.Fatal(err)
		}
		got = append(got, message)
	}
	if len(got) != 6 {
		t.Fatalf("got %d output messages, want 6: %#v", len(got), got)
	}

	initializeResult := got[0]["result"].(map[string]any)
	capabilities := initializeResult["capabilities"].(map[string]any)
	if capabilities["documentFormattingProvider"] != true || capabilities["textDocumentSync"] != float64(1) {
		t.Fatalf("unexpected capabilities: %#v", capabilities)
	}
	if capabilities["semanticTokensProvider"] == nil {
		t.Fatalf("semantic token capability missing: %#v", capabilities)
	}
	if capabilities["signatureHelpProvider"] == nil {
		t.Fatalf("signature help capability missing: %#v", capabilities)
	}
	if capabilities["renameProvider"] == nil {
		t.Fatalf("rename capability missing: %#v", capabilities)
	}

	firstDiagnostics := got[1]["params"].(map[string]any)["diagnostics"].([]any)
	if len(firstDiagnostics) != 0 {
		t.Fatalf("valid document produced diagnostics: %#v", firstDiagnostics)
	}

	edits := got[2]["result"].([]any)
	if len(edits) != 1 {
		t.Fatalf("formatting edits = %#v", edits)
	}
	newText := edits[0].(map[string]any)["newText"]
	wantText := "IF (up.x == 1)\n  Log(\"x\")\nENDIF\n"
	if newText != wantText {
		t.Fatalf("formatted text = %q, want %q", newText, wantText)
	}

	semanticData := got[3]["result"].(map[string]any)["data"].([]any)
	if len(semanticData) == 0 || len(semanticData)%5 != 0 {
		t.Fatalf("invalid semantic tokens: %#v", semanticData)
	}

	changedDiagnostics := got[4]["params"].(map[string]any)["diagnostics"].([]any)
	if len(changedDiagnostics) == 0 {
		t.Fatal("invalid changed document produced no diagnostics")
	}
}
