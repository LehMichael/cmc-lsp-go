package lsp

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/lehmichael/cmc-lsp-go/internal/workspace"
)

func TestRenameCallableAcrossProject(t *testing.T) {
	root := t.TempDir()
	projectDirectory := filepath.Join(root, "Project")
	libraryDirectory := filepath.Join(root, "Database", "Library")
	scriptDirectory := filepath.Join(root, "Database", "Scripts")
	for _, directory := range []string{projectDirectory, libraryDirectory, scriptDirectory} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	library := filepath.Join(libraryDirectory, "common.uplib")
	script := filepath.Join(scriptDirectory, "main.upscr")
	if err := os.WriteFile(library, []byte("func Helper($) {\n    Return(Up.$1)\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(script, []byte("helper(1)\nUp.Helper = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	projectXML := `<?xml version="1.0"?><pack><config><list name="ScriptLibList"><dlink ref="..\Database\Library\common.uplib" /></list></config><script ref="..\Database\Scripts\main.upscr" /></pack>`
	if err := os.WriteFile(filepath.Join(projectDirectory, "machine.upproj"), []byte(projectXML), 0o644); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	server := NewLsp(bytes.NewReader(nil), &output)
	rootURI := workspace.PathToURI(root)
	server.loadProjects(initializeParams{RootURI: &rootURI})
	scriptURI := workspace.PathToURI(script)
	libraryURI := workspace.PathToURI(library)
	position := map[string]any{"line": 0, "character": 2}

	server.handlePrepareRename(requestMessage{ID: intMessageID(1), Params: marshalParams(t, map[string]any{
		"textDocument": map[string]any{"uri": scriptURI}, "position": position,
	})})
	var prepared struct {
		Result struct {
			Range       lspRange `json:"range"`
			Placeholder string   `json:"placeholder"`
		} `json:"result"`
	}
	readResponse(t, &output, &prepared)
	if prepared.Result.Placeholder != "helper" || prepared.Result.Range != (lspRange{Start: positionValue(0, 0), End: positionValue(0, 6)}) {
		t.Fatalf("prepare rename = %#v", prepared.Result)
	}

	server.handleRename(requestMessage{ID: intMessageID(2), Params: marshalParams(t, map[string]any{
		"textDocument": map[string]any{"uri": scriptURI}, "position": position, "newName": "ReadMachineValue",
	})})
	var renamed struct {
		Result workspaceEdit `json:"result"`
	}
	readResponse(t, &output, &renamed)
	if len(renamed.Result.Changes) != 2 || len(renamed.Result.Changes[scriptURI]) != 1 || len(renamed.Result.Changes[libraryURI]) != 1 {
		t.Fatalf("workspace edit = %#v", renamed.Result)
	}
	for uri, edits := range renamed.Result.Changes {
		for _, edit := range edits {
			if edit.NewText != "ReadMachineValue" {
				t.Fatalf("edit for %s = %#v", uri, edit)
			}
		}
	}
}

func TestRenameRejectsUnsafeTargets(t *testing.T) {
	const uri = "file:///tmp/rename.upscr"
	const source = "Log(1)\nUp.value = 1\n"
	server := NewLsp(bytes.NewReader(nil), &bytes.Buffer{})
	if err := server.overlay.Open(uri, source, 1); err != nil {
		t.Fatal(err)
	}
	for _, target := range []position{{Line: 0, Character: 1}, {Line: 1, Character: 4}} {
		occurrence, definition := server.renameTarget(uri, target)
		if occurrence != nil || definition != nil {
			t.Fatalf("rename target at %#v = %#v, %#v", target, occurrence, definition)
		}
	}
	for _, name := range []string{"", "1Helper", "Up.Helper", "Helper-name", "$MD123"} {
		if isSimpleIdentifier(name) {
			t.Errorf("isSimpleIdentifier(%q) = true", name)
		}
	}
}

func positionValue(line, character int) position {
	return position{Line: line, Character: character}
}
