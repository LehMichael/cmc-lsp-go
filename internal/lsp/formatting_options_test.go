package lsp

import (
	"bytes"
	"testing"
)

func TestFormattingAlignmentOptions(t *testing.T) {
	const uri = "file:///tmp/alignment.upscr"
	const source = "Up.x=1 ; one\nUp.longName=22 ; two\n"
	var output bytes.Buffer
	server := NewLsp(bytes.NewReader(nil), &output)
	if err := server.overlay.Open(uri, source, 1); err != nil {
		t.Fatal(err)
	}

	server.handleFormatting(requestMessage{ID: intMessageID(1), Params: marshalParams(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"options":      map[string]any{"tabSize": 4, "insertSpaces": true},
	})})
	var aligned struct {
		Result []textEdit `json:"result"`
	}
	readResponse(t, &output, &aligned)
	if len(aligned.Result) != 1 || aligned.Result[0].NewText != "Up.x        = 1   ; one\nUp.longName = 22  ; two\n" {
		t.Fatalf("default aligned formatting = %#v", aligned.Result)
	}

	server.handleFormatting(requestMessage{ID: intMessageID(2), Params: marshalParams(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"options": map[string]any{
			"tabSize": 4, "insertSpaces": true,
			"cmcAlignConsecutiveAssignments": false,
			"cmcAlignTrailingComments":       false,
		},
	})})
	var unaligned struct {
		Result []textEdit `json:"result"`
	}
	readResponse(t, &output, &unaligned)
	if len(unaligned.Result) != 1 || unaligned.Result[0].NewText != "Up.x = 1  ; one\nUp.longName = 22  ; two\n" {
		t.Fatalf("disabled alignment formatting = %#v", unaligned.Result)
	}
}
