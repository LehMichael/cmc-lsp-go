package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestBuiltinCallableCatalogCoversManualSection(t *testing.T) {
	want := []string{
		"CHANDATA",
		"SinamicsTecActivate", "SinamicsTecDeactivate", "IsBit", "SetBit", "ClrBit",
		"StringLen", "StringMatch", "StringPos", "StringReplace", "StringSubStr",
		"FileCopy", "FileDelete", "FileExist", "FileRead", "FileWrite", "QueryIni", "QueryXml", "TraceToFile",
		"AddXmlElement", "InsertXmlElement", "GetXmlElement", "GetXmlElementText", "GetXmlAttribute", "SetXmlElementText", "SetXmlAttribute", "RemoveXmlElement", "RemoveXmlAttribute",
		"ConfigDataItemExists", "ReadConfigDataItem", "SetConfigDataItem", "RemoveConfigDataItem",
		"Msg", "Warning", "Error", "Input", "InputChoice", "InputEnum", "InputInt", "InputReal", "InputText", "InputUInt",
		"ResFile", "ResText",
		"Skip", "Redo", "Return", "ExtCall", "DateTime", "DOVar", "Log", "Logging", "MathRound", "Version",
		"Prepare", "Patch",
	}
	if len(builtinCallables) != len(want) {
		t.Fatalf("catalog contains %d callables, want %d", len(builtinCallables), len(want))
	}
	for _, name := range want {
		hover, ok := builtinCallableHover(name, "en-US")
		if !ok {
			t.Errorf("missing hover for %s", name)
			continue
		}
		for _, part := range []string{"### `" + name, "Source: Siemens Create MyConfig manual"} {
			if !strings.Contains(hover, part) {
				t.Errorf("hover for %s = %q, missing %q", name, hover, part)
			}
		}
	}
}

func TestBuiltinCallableHoverUsesClientLocale(t *testing.T) {
	tests := []struct {
		name   string
		locale string
		want   []string
	}{
		{name: "ExtCall", locale: "en-US", want: []string{"Procedure", "external UTF-8 manipulation task", "section 7.8.17.4"}},
		{name: "filewrite", locale: "de-AT", want: []string{"Prozedur", "Schreibt eine Zeichenkette", "section 7.8.14.5"}},
		{name: "SetXmlElementText", locale: "de-DE", want: []string{"Funktion", "Schreibt Text", "section 6.16.6"}},
		{name: "RemoveXmlElement", locale: "en-US", want: []string{"RemoveXmlElement(<area>", "Removes every XML element", "signature corrected"}},
		{name: "SetBit", locale: "de-AT", want: []string{"Standardbibliotheksfunktion", "Bit auf `1`", "section 6.6.2.3"}},
		{name: "Match", locale: "de-DE", want: []string{"Match(\"<string>\"", "Veralteter Kompatibilit\u00e4tsname", "`StringMatch`"}},
	}
	for _, test := range tests {
		hover, ok := builtinCallableHover(test.name, test.locale)
		if !ok {
			t.Fatalf("builtinCallableHover(%q, %q) returned no hover", test.name, test.locale)
		}
		for _, want := range test.want {
			if !strings.Contains(hover, want) {
				t.Errorf("builtinCallableHover(%q, %q) = %q, missing %q", test.name, test.locale, hover, want)
			}
		}
	}
}

func TestBuiltinCallableCompletionIncludesAllCurrentNames(t *testing.T) {
	items := builtinCompletionItems()
	if len(items) != len(builtinCallables) {
		t.Fatalf("got %d completion items, want %d", len(items), len(builtinCallables))
	}
	labels := make(map[string]bool, len(items))
	for _, item := range items {
		labels[item.Label] = true
		if item.Documentation == nil || item.Documentation.Kind != "markdown" {
			t.Errorf("completion %s has no markdown documentation", item.Label)
		}
	}
	for _, name := range []string{"ExtCall", "FileWrite", "QueryIni", "ResText", "Prepare", "GetXmlElementText", "SetConfigDataItem", "IsBit"} {
		if !labels[name] {
			t.Errorf("missing completion for %s", name)
		}
	}
	for _, alias := range []string{"Match", "Replace", "Exists", "Round"} {
		if labels[alias] {
			t.Errorf("deprecated alias %s should not be offered as a completion", alias)
		}
	}
}

func TestBuiltinCallableHoverRequest(t *testing.T) {
	const uri = "file:///tmp/builtins.upscr"
	const source = `ExtCall("./Data_X_Axis.TEA")`
	var output bytes.Buffer
	server := NewLsp(bytes.NewReader(nil), &output)
	server.locale = "de-AT"
	if err := server.overlay.Open(uri, source, 1); err != nil {
		t.Fatal(err)
	}
	params, err := json.Marshal(map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 0, "character": 3},
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
			Contents markupContent `json:"contents"`
		} `json:"result"`
	}
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`ExtCall("<path>")`, "Prozedur", "externen UTF-8-Manipulationsauftrag"} {
		if !strings.Contains(response.Result.Contents.Value, want) {
			t.Errorf("hover = %q, missing %q", response.Result.Contents.Value, want)
		}
	}
}
