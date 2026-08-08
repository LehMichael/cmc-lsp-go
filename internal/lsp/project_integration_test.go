package lsp

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/lehmichael/cmc-lsp-go/internal/workspace"
)

func TestProjectIndexUsesUPProjScope(t *testing.T) {
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
	unreferenced := filepath.Join(scriptDirectory, "unreferenced.upscr")
	if err := os.WriteFile(library, []byte("func Helper($) {\n    Return(Up.$1)\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(script, []byte("Up.result = Helper(1)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unreferenced, []byte("Helper(2)\n"), 0o644); err != nil {
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
	if len(server.projects) != 1 {
		t.Fatalf("loaded %d projects, want 1", len(server.projects))
	}
	scriptURI := workspace.PathToURI(script)
	definition := server.definitionAt(scriptURI, "Helper")
	if definition == nil || definition.URI != workspace.PathToURI(library) {
		t.Fatalf("Helper definition = %#v", definition)
	}
	if definition := server.definitionAt(workspace.PathToURI(unreferenced), "Helper"); definition != nil {
		t.Fatalf("unreferenced single-file script leaked project definition: %#v", definition)
	}

	references := 0
	for _, occurrence := range server.scopedOccurrences(scriptURI) {
		if occurrence.Name == "Helper" {
			references++
		}
	}
	if references != 2 {
		t.Fatalf("Helper occurrences = %d, want definition and call", references)
	}
}
