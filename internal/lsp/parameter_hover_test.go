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

func TestParameterDatabaseHover(t *testing.T) {
	directory := t.TempDir()
	dataDirectory := filepath.Join(directory, "en")
	if err := os.Mkdir(dataDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	contents := `<?xml version="1.0"?><info><parameter number="10000" type="STRING" dim="1"><name>AXCONF_MACHAX_NAME_TAB</name><brief>Machine axis name</brief><description>List of machine axis identifiers.</description></parameter></info>`
	if err := os.WriteFile(filepath.Join(dataDirectory, "mdnck.mdat"), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	catalog, err := database.Load(directory, "en-US")
	if err != nil {
		t.Fatal(err)
	}

	const uri = "file:///tmp/parameter-hover.tea"
	const source = "Up.name = $MN_AXCONF_MACHAX_NAME_TAB\n"
	var output bytes.Buffer
	server := NewLsp(bytes.NewReader(nil), &output)
	server.parameters = catalog
	if err := server.overlay.Open(uri, source, 1); err != nil {
		t.Fatal(err)
	}
	params, err := json.Marshal(map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 0, "character": 15},
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
	if !strings.Contains(response.Result.Contents.Value, "Machine axis name") ||
		!strings.Contains(response.Result.Contents.Value, "List of machine axis identifiers") {
		t.Fatalf("hover = %q", response.Result.Contents.Value)
	}
}

func TestLocalParameterDatabaseHover(t *testing.T) {
	directory := filepath.Join("..", "..", "DataBase")
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		t.Skip("local Siemens DataBase directory is not available")
	} else if err != nil {
		t.Fatal(err)
	}
	catalog, err := database.Load(directory, "de-DE")
	if err != nil {
		t.Fatal(err)
	}
	hover, ok := catalog.Hover("$MA_NUM_ENCS")
	if !ok || hover == "" {
		t.Fatalf("local machine-data hover = %q, %v", hover, ok)
	}
}
