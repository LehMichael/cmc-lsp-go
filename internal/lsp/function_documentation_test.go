package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

const documentedFunctionSource = `;Description: Example of a function definition
;Arg1:<string>
;Arg2:<doVar>
func MyFunction($, $) {
  Return(0)
}

MyFunction("value", Up.drive)
`

func TestDocumentedFunctionCompletion(t *testing.T) {
	const uri = "file:///tmp/documented.uplib"
	var output bytes.Buffer
	server := NewLsp(bytes.NewReader(nil), &output)
	if err := server.overlay.Open(uri, documentedFunctionSource, 1); err != nil {
		t.Fatal(err)
	}
	params, err := json.Marshal(map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 7, "character": 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	server.handleCompletion(requestMessage{ID: intMessageID(1), Method: "textDocument/completion", Params: params})

	payload, err := readFrame(bufio.NewReader(&output))
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		Result []completionItem `json:"result"`
	}
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatal(err)
	}
	for _, item := range response.Result {
		if item.Label != "MyFunction" {
			continue
		}
		if item.Detail != "func(<string>, <doVar>)" || item.Documentation == nil || item.Documentation.Kind != "markdown" || !strings.Contains(item.Documentation.Value, "Example of a function definition") || !strings.Contains(item.Documentation.Value, "Arg1: `<string>`") {
			t.Fatalf("completion = %#v", item)
		}
		return
	}
	t.Fatal("documented function completion not found")
}

func TestDocumentedFunctionHover(t *testing.T) {
	const uri = "file:///tmp/documented.uplib"
	var output bytes.Buffer
	server := NewLsp(bytes.NewReader(nil), &output)
	if err := server.overlay.Open(uri, documentedFunctionSource, 1); err != nil {
		t.Fatal(err)
	}
	params, err := json.Marshal(map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 7, "character": 3},
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
	wantParts := []string{"func MyFunction(<string>, <doVar>)", "Example of a function definition", "Arg2: `<doVar>`"}
	for _, want := range wantParts {
		if !strings.Contains(response.Result.Contents.Value, want) {
			t.Fatalf("hover = %q, missing %q", response.Result.Contents.Value, want)
		}
	}
}
