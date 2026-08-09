package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestSystemVariableHoverDocumentation(t *testing.T) {
	tests := []struct {
		identifier string
		locale     string
		want       string
	}{
		{"Up.$Pack.NCU", "en-US", "NCU/PPU data area"},
		{"Up.$Pack.NCU", "de-DE", "NCU/PPU-Datenbereich"},
		{"Up.$Step[Axis1].Processing", "en-US", "green execution track"},
		{"Up.$Dialog.ArcSelection.ArchiveIn", "en-US", "input archive"},
		{"Up.$Dialog.NewerDialog.Activated", "en-US", "processing of this dialog page"},
	}
	for _, test := range tests {
		hover, ok := systemVariableHover(test.identifier, test.locale)
		if !ok || !strings.Contains(hover, test.want) {
			t.Errorf("systemVariableHover(%q, %q) = %q, %v", test.identifier, test.locale, hover, ok)
		}
	}
}

func TestSystemVariableAt(t *testing.T) {
	const source = "If Up.$Step[Axis1].Activated == true\n"
	if got := systemVariableAt(source, position{Line: 0, Character: 20}); got != "Up.$Step[Axis1].Activated" {
		t.Fatalf("systemVariableAt = %q", got)
	}
}

func TestHandleSystemVariableHover(t *testing.T) {
	const uri = "file:///tmp/system-variable.upact"
	const source = "If Up.$Env.BatchMode == true\n"
	var output bytes.Buffer
	server := NewLsp(bytes.NewReader(nil), &output)
	server.locale = "de-AT"
	if err := server.overlay.Open(uri, source, 1); err != nil {
		t.Fatal(err)
	}
	params, err := json.Marshal(map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 0, "character": 12},
	})
	if err != nil {
		t.Fatal(err)
	}
	server.handleHover(requestMessage{ID: intMessageID(1), Method: "textDocument/hover", Params: params})
	payload, err := readFrame(bufio.NewReader(&output))
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		Result struct {
			Contents struct {
				Value string `json:"value"`
			} `json:"contents"`
		} `json:"result"`
	}
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(response.Result.Contents.Value, "Kommandozeilen-Batchmodus") {
		t.Fatalf("hover = %q", response.Result.Contents.Value)
	}
}
