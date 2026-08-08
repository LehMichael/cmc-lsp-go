package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	directory := t.TempDir()
	projectDirectory := filepath.Join(directory, "Project")
	libraryDirectory := filepath.Join(directory, "Database", "Library")
	if err := os.MkdirAll(projectDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(libraryDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	library := filepath.Join(libraryDirectory, "common.uplib")
	script := filepath.Join(projectDirectory, "start.upscr")
	if err := os.WriteFile(library, []byte("proc Common() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(script, []byte("Common()\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	projectPath := filepath.Join(projectDirectory, "machine.upproj")
	xml := `<?xml version="1.0"?><pack version="80" appVersion="6.9.0.0"><config><list name="ScriptLibList"><dlink ref="..\Database\Library\common.uplib" /></list></config><event><script ref="start.upscr" /></event></pack>`
	if err := os.WriteFile(projectPath, append([]byte{0xEF, 0xBB, 0xBF}, []byte(xml)...), 0o644); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(projectPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version != "80" || loaded.AppVersion != "6.9.0.0" {
		t.Fatalf("metadata = %#v", loaded)
	}
	if len(loaded.Libraries) != 1 || loaded.Libraries[0] != library {
		t.Fatalf("libraries = %#v", loaded.Libraries)
	}
	if len(loaded.Scripts) != 1 || loaded.Scripts[0] != script {
		t.Fatalf("scripts = %#v", loaded.Scripts)
	}
	if len(loaded.Missing) != 0 || !loaded.Contains(script) {
		t.Fatalf("unexpected project graph: %#v", loaded)
	}
}
