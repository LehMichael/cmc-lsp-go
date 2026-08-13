package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lehmichael/cmc-lsp-go/internal/database"
)

func TestUpMemberCompletionContainsOnlyUpMembers(t *testing.T) {
	const uri = "file:///tmp/up-completion.upscr"
	const source = "Up.CustomValue = 1\nUp."
	server, output := completionTestServer(t, uri, source, nil)

	requestCompletion(t, server, uri, documentEnd(source))
	var response struct {
		Result []completionItem `json:"result"`
	}
	readCompletionResponse(t, output, &response)

	labels := completionLabels(response.Result)
	for _, want := range []string{"$Dialog", "$Env", "$Pack", "$Step", "CustomValue"} {
		if !labels[want] {
			t.Errorf("missing Up member %q in %#v", want, response.Result)
		}
	}
	for _, unwanted := range []string{"#include", "If", "$MN_MM_NUM_TOOL"} {
		if labels[unwanted] {
			t.Errorf("unexpected global completion %q in Up member context", unwanted)
		}
	}
}

func TestNestedUpMemberCompletionContainsOnlyDirectMembers(t *testing.T) {
	const uri = "file:///tmp/nested-up-completion.upscr"
	const source = "Up.Group.Child = 1\nUp.Group.Grandchild.Value = 2\nUp.Group."
	server, output := completionTestServer(t, uri, source, nil)

	requestCompletion(t, server, uri, documentEnd(source))
	var response struct {
		Result []completionItem `json:"result"`
	}
	readCompletionResponse(t, output, &response)

	labels := completionLabels(response.Result)
	if !labels["Child"] || !labels["Grandchild"] || labels["Grandchild.Value"] || labels["If"] {
		t.Fatalf("nested Up completion = %#v", response.Result)
	}
}

func TestParameterCompletionUsesDatabaseCatalog(t *testing.T) {
	catalog := testCompletionCatalog(t)
	tests := []struct {
		prefix     string
		wantLabel  string
		wantDetail string
	}{
		{"$MN", "$MN_AXCONF_MACHAX_NAME_TAB", "Machine data · STRING — Machine axis name"},
		{"P", "p10", "SINAMICS write parameter · Int16 — Writable drive value"},
		{"R", "r2", "SINAMICS read parameter · Int16 — Read-only drive value"},
	}
	for _, test := range tests {
		t.Run(test.prefix, func(t *testing.T) {
			const uri = "file:///tmp/parameter-completion.tea"
			server, output := completionTestServer(t, uri, test.prefix, catalog)
			requestCompletion(t, server, uri, documentEnd(test.prefix))
			var response struct {
				Result completionList `json:"result"`
			}
			readCompletionResponse(t, output, &response)
			if len(response.Result.Items) != 1 {
				t.Fatalf("completion for %q = %#v", test.prefix, response.Result.Items)
			}
			item := response.Result.Items[0]
			if item.Label != test.wantLabel || item.Detail != test.wantDetail {
				t.Fatalf("completion for %q = %#v", test.prefix, item)
			}
			if item.Documentation == nil || !strings.Contains(item.Documentation.Value, item.Label) {
				t.Errorf("completion documentation for %q = %#v", test.prefix, item.Documentation)
			}
		})
	}
}

func TestIdentifierDetailOnlyClassifiesUpVariablesAsCMCVariables(t *testing.T) {
	if got := identifierDetail("Up.Value"); got != "CMC data or package variable" {
		t.Errorf("Up detail = %q", got)
	}
	if got := identifierDetail("$MN_TEST"); got != "CMC parameter" {
		t.Errorf("machine-data detail = %q", got)
	}
	if got := identifierDetail("p10"); got != "SINAMICS parameter" {
		t.Errorf("drive-parameter detail = %q", got)
	}
	if got := identifierDetail("LocalValue"); got != "Variable" {
		t.Errorf("local variable detail = %q", got)
	}
}

func completionTestServer(t *testing.T, uri, source string, catalog *database.Catalog) (*Lsp, *bytes.Buffer) {
	t.Helper()
	output := &bytes.Buffer{}
	server := NewLsp(bytes.NewReader(nil), output)
	server.parameters = catalog
	if err := server.overlay.Open(uri, source, 1); err != nil {
		t.Fatal(err)
	}
	return server, output
}

func requestCompletion(t *testing.T, server *Lsp, uri string, target position) {
	t.Helper()
	params, err := json.Marshal(map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     target,
	})
	if err != nil {
		t.Fatal(err)
	}
	server.handleCompletion(requestMessage{ID: intMessageID(1), Method: "textDocument/completion", Params: params})
}

func readCompletionResponse(t *testing.T, output *bytes.Buffer, response any) {
	t.Helper()
	payload, err := readFrame(bufio.NewReader(output))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, response); err != nil {
		t.Fatal(err)
	}
}

func completionLabels(items []completionItem) map[string]bool {
	labels := make(map[string]bool, len(items))
	for _, item := range items {
		labels[item.Label] = true
	}
	return labels
}

func testCompletionCatalog(t *testing.T) *database.Catalog {
	t.Helper()
	root := t.TempDir()
	english := filepath.Join(root, "en")
	if err := os.Mkdir(english, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"mdnck.mdat": `<?xml version="1.0"?><info><parameter number="10000" type="STRING" dim="1"><name>AXCONF_MACHAX_NAME_TAB</name><brief>Machine axis name</brief><description>List of machine axis identifiers.</description></parameter></info>`,
		"do001.para": `<?xml version="1.0"?><info comment="DEVICE"><parameter number="10" type="Int16" dim="0"><name>Test</name><brief>Writable drive value</brief></parameter><parameter number="2" type="Int16" dim="0" readonly="true"><name>Status</name><brief>Read-only drive value</brief></parameter></info>`,
	}
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(english, name), []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	catalog, err := database.Load(root, "en-US")
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}
