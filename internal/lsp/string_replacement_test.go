package lsp

import (
	"bytes"
	"strings"
	"testing"
)

func TestReferencesIndexStringReplacements(t *testing.T) {
	const uri = "file:///tmp/string-replacement.upscr"
	const source = "Up.path = \"base\"\nUp.result = \"$(Up.path)/file.tea\"\n"
	var output bytes.Buffer
	server := NewLsp(bytes.NewReader(nil), &output)
	if err := server.overlay.Open(uri, source, 1); err != nil {
		t.Fatal(err)
	}
	line := strings.Split(source, "\n")[1]
	server.handleReferences(requestMessage{ID: intMessageID(1), Params: marshalParams(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position": map[string]any{
			"line": 1, "character": strings.LastIndex(line, "path") + 1,
		},
	})})
	var response struct {
		Result []location `json:"result"`
	}
	readResponse(t, &output, &response)
	assertReferenceLines(t, response.Result, []int{0, 1})
}
