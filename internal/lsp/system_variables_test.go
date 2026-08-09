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
		{"Up.$Pack.HmiDataHandlings.UseOperateNodeInActions", "en-US", "Operate nodes"},
		{"Up.$Dialog.AccessData.Targets.IPC", "en-US", "NCU and PCU"},
		{"Up.$Dialog.NcuSetup.Modes.SOFTWAREONLY", "de-DE", "Nur Zusatzsoftware"},
		{"Up.$Dialog.PlcConfig.ConfigDataItemsSources.UPDATE", "en-US", "selected file"},
		{"Up.$Dialog.SystemConfig.NcSources.FACTORY", "en-US", "general reset"},
		{"Up.$Dialog.SystemConfig.DrvSources.TARGET", "de-DE", "Benutzerdefinierte Topologie"},
		{"Up.$Dialog.NcuOrigin.ConfigDataItemsSelections.FILTERED", "en-US", "XML filter file"},
		{"Up.$Dialog.ArcFileInstall.Entry[German].Install", "en-US", "language archive is installed"},
		{"Up.$AccessLevelPWDConfig.ManufactFile", "en-US", "manufacturer access-level password"},
		{"Up.$BasicSecSettings.SecArchPasswordFile", "de-DE", "Security-Archiv"},
		{"Up.$Step[Axis1].Processing", "en-US", "whether the step was executed"},
		{"Up.$Dialog.PackageConfig.Step[Axis1].Processing", "en-US", "dialog step was executed"},
		{"Up.$Env.SDID", "de-DE", "SD-Kartenkennung"},
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

func TestCurrentSystemVariableDocumentationIsComplete(t *testing.T) {
	for name, documentation := range currentSystemVariableDocumentationByName {
		if name != strings.ToLower(name) {
			t.Errorf("system variable key %q is not canonical", name)
		}
		if documentation.Type == "" || documentation.English == "" || documentation.German == "" || documentation.Manual == "" {
			t.Errorf("documentation for %q is incomplete: %#v", name, documentation)
		}
	}
}

func TestCanonicalSystemVariableNormalizesIndexes(t *testing.T) {
	tests := map[string]string{
		"Up.$Step[Axis1].Activated":                               "up.$step[?].activated",
		"Up.$Dialog.ArcFileInstall.Entry[$(Up.EntryID)].File":     "up.$dialog.arcfileinstall.entry[?].file",
		"Up.$Dialog.PackageConfig.Step[PackageConfig].Processing": "up.$dialog.packageconfig.step[?].processing",
	}
	for identifier, want := range tests {
		if got := canonicalSystemVariable(identifier); got != want {
			t.Errorf("canonicalSystemVariable(%q) = %q, want %q", identifier, got, want)
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
