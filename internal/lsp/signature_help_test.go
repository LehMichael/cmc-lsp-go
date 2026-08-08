package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"testing"
)

func TestDocumentedFunctionSignatureHelp(t *testing.T) {
	const uri = "file:///tmp/signature.uplib"
	const source = `;Description: Combines a value with a drive object
;Arg1:<string>
;Arg2:<doVar>
func MyFunction($, $) {
    Return(0)
}

MyFunction(StringLen("x,y"), Up.drive)
`
	var output bytes.Buffer
	server := NewLsp(bytes.NewReader(nil), &output)
	if err := server.overlay.Open(uri, source, 1); err != nil {
		t.Fatal(err)
	}
	params := marshalParams(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 7, "character": 36},
	})
	server.handleSignatureHelp(requestMessage{ID: intMessageID(1), Params: params})

	var response struct {
		Result signatureHelp `json:"result"`
	}
	readResponse(t, &output, &response)
	if len(response.Result.Signatures) != 1 {
		t.Fatalf("signatures = %#v", response.Result.Signatures)
	}
	signature := response.Result.Signatures[0]
	if signature.Label != "MyFunction(<string>, <doVar>)" {
		t.Fatalf("label = %q", signature.Label)
	}
	if response.Result.ActiveParameter != 1 {
		t.Fatalf("active parameter = %d, want 1", response.Result.ActiveParameter)
	}
	if signature.Documentation == nil || signature.Documentation.Value != "Combines a value with a drive object" {
		t.Fatalf("documentation = %#v", signature.Documentation)
	}
	if len(signature.Parameters) != 2 || signature.Parameters[1].Documentation == nil || signature.Parameters[1].Documentation.Value != "<doVar>" {
		t.Fatalf("parameters = %#v", signature.Parameters)
	}
}

func TestSignatureHelpIgnoresCommentsAndStrings(t *testing.T) {
	for _, test := range []struct {
		name      string
		source    string
		character int
	}{
		{name: "comment", source: `; MyFunction(1, 2)`, character: 15},
		{name: "string", source: `Up.value = "MyFunction(1, 2)"`, character: 24},
	} {
		t.Run(test.name, func(t *testing.T) {
			const uri = "file:///tmp/no-signature.upscr"
			text := "func MyFunction($, $) {\n}\n" + test.source
			var output bytes.Buffer
			server := NewLsp(bytes.NewReader(nil), &output)
			if err := server.overlay.Open(uri, text, 1); err != nil {
				t.Fatal(err)
			}
			params := marshalParams(t, map[string]any{
				"textDocument": map[string]any{"uri": uri},
				"position":     map[string]any{"line": 2, "character": test.character},
			})
			server.handleSignatureHelp(requestMessage{ID: intMessageID(1), Params: params})
			var response struct {
				Result *signatureHelp `json:"result"`
			}
			readResponse(t, &output, &response)
			if response.Result != nil {
				t.Fatalf("signature help = %#v, want nil", response.Result)
			}
		})
	}
}

func TestSignatureHelpIgnoresDefinitionHeader(t *testing.T) {
	const uri = "file:///tmp/definition.uplib"
	const source = "func MyFunction($, $) {\n}\n"
	var output bytes.Buffer
	server := NewLsp(bytes.NewReader(nil), &output)
	if err := server.overlay.Open(uri, source, 1); err != nil {
		t.Fatal(err)
	}
	params := marshalParams(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 0, "character": 18},
	})
	server.handleSignatureHelp(requestMessage{ID: intMessageID(1), Params: params})
	var response struct {
		Result *signatureHelp `json:"result"`
	}
	readResponse(t, &output, &response)
	if response.Result != nil {
		t.Fatalf("signature help = %#v, want nil", response.Result)
	}
}

func marshalParams(t *testing.T, value any) json.RawMessage {
	t.Helper()
	params, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return params
}

func readResponse(t *testing.T, output *bytes.Buffer, target any) {
	t.Helper()
	payload, err := readFrame(bufio.NewReader(output))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		t.Fatal(err)
	}
}
