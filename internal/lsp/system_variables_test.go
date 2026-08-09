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
		if documentation.Name == "" || documentation.Type == "" || documentation.English == "" || documentation.German == "" || documentation.Manual == "" {
			t.Errorf("documentation for %q is incomplete: %#v", name, documentation)
		}
	}
}

func TestSystemVariableMemberCompletion(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		locale     string
		wantLabels []string
		wantDetail string
	}{
		{
			name:       "all step members",
			source:     "If Up.$Step[1].",
			locale:     "en-US",
			wantLabels: []string{"Activated", "Collapsed", "Locked", "Processing"},
		},
		{
			name:       "partial step member",
			source:     "If (Up.$STEP[1].Pro",
			locale:     "en-US",
			wantLabels: []string{"Processing"},
			wantDetail: "BOOL \u00b7 Read-only \u2014 Runtime feedback indicating whether the step was executed.",
		},
		{
			name:       "dialog step",
			source:     "Up.$Dialog.PackageConfig.Step[IPC].L",
			locale:     "en-US",
			wantLabels: []string{"Locked"},
		},
		{
			name:       "indexed archive entry",
			source:     "Up.$Dialog.ArcFileInstall.Entry[German].I",
			locale:     "en-US",
			wantLabels: []string{"Install"},
		},
		{
			name:       "localized enum member",
			source:     "Up.$Dialog.NcuSetup.Modes.S",
			locale:     "de-AT",
			wantLabels: []string{"SOFTWAREONLY"},
			wantDetail: "Mode \u00b7 Schreibgesch\u00fctzt \u2014 Nur Zusatzsoftware installieren.",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items, contextual := systemVariableCompletionItems(test.source, documentEnd(test.source), test.locale)
			if !contextual {
				t.Fatal("completion was not recognized as a system-variable member context")
			}
			labels := make([]string, len(items))
			for index, item := range items {
				labels[index] = item.Label
				if item.Documentation == nil || !strings.Contains(item.Documentation.Value, "Siemens Create MyConfig manual") {
					t.Errorf("completion %q has no manual documentation: %#v", item.Label, item.Documentation)
				}
			}
			if strings.Join(labels, ",") != strings.Join(test.wantLabels, ",") {
				t.Fatalf("labels = %q, want %q", labels, test.wantLabels)
			}
			if test.wantDetail != "" && items[0].Detail != test.wantDetail {
				t.Errorf("detail = %q, want %q", items[0].Detail, test.wantDetail)
			}
		})
	}
}

func TestHandleSystemVariableMemberCompletionSuppressesGlobalItems(t *testing.T) {
	const uri = "file:///tmp/member-completion.upact"
	const source = "If (Up.$STEP[1].Pro"
	var output bytes.Buffer
	server := NewLsp(bytes.NewReader(nil), &output)
	if err := server.overlay.Open(uri, source, 1); err != nil {
		t.Fatal(err)
	}
	params, err := json.Marshal(map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     documentEnd(source),
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
	if len(response.Result) != 1 || response.Result[0].Label != "Processing" {
		t.Fatalf("completion = %#v", response.Result)
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
	tests := []struct {
		source string
		column int
		want   string
	}{
		{"If Up.$Step[Axis1].Activated == true\n", 20, "Up.$Step[Axis1].Activated"},
		{"If (Up.$Step[Axis1].Activated)\n", 21, "Up.$Step[Axis1].Activated"},
		{"If (Up.$Step[$(Up.StepID)].Processing == true)\n", 31, "Up.$Step[$(Up.StepID)].Processing"},
	}
	for _, test := range tests {
		if got := systemVariableAt(test.source, position{Line: 0, Character: test.column}); got != test.want {
			t.Errorf("systemVariableAt(%q) = %q, want %q", test.source, got, test.want)
		}
	}
}

func TestHandleSystemVariableHoverInsideParenthesizedIf(t *testing.T) {
	const uri = "file:///tmp/system-variable.upact"
	const source = "If (Up.$Env.BatchMode == true)\n"
	var output bytes.Buffer
	server := NewLsp(bytes.NewReader(nil), &output)
	server.locale = "de-AT"
	if err := server.overlay.Open(uri, source, 1); err != nil {
		t.Fatal(err)
	}
	params, err := json.Marshal(map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     map[string]any{"line": 0, "character": 13},
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
