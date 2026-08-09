package lsp

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/lehmichael/cmc-lsp-go/internal/workspace"
)

func TestSingleFileDiscoversAdjacentAndConfiguredLibraries(t *testing.T) {
	root := t.TempDir()
	scriptDirectory := filepath.Join(root, "Scripts")
	firstLibraryDirectory := filepath.Join(root, "Libraries", "First")
	secondLibraryDirectory := filepath.Join(root, "Libraries", "Second")
	for _, directory := range []string{scriptDirectory, firstLibraryDirectory, secondLibraryDirectory} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	script := filepath.Join(scriptDirectory, "main.upscr")
	adjacent := filepath.Join(scriptDirectory, "adjacent.uplib")
	first := filepath.Join(firstLibraryDirectory, "first.uplib")
	second := filepath.Join(secondLibraryDirectory, "second.uplib")
	ignored := filepath.Join(firstLibraryDirectory, "ignored.upscr")
	files := map[string]string{
		script:   "Adjacent()\nFirst()\nSecond()\nShared()\nIgnored()\n",
		adjacent: "proc Adjacent() {\n}\nproc Shared() {\n}\n",
		first:    "proc First() {\n}\nproc Shared() {\n}\n",
		second:   "proc Second() {\n}\nproc Shared() {\n}\n",
		ignored:  "proc Ignored() {\n}\n",
	}
	for path, text := range files {
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	server := NewLsp(bytes.NewReader(nil), &bytes.Buffer{})
	rootURI := workspace.PathToURI(root)
	params := initializeParams{RootURI: &rootURI}
	params.InitializationOptions.CMCLibraryPath = firstLibraryDirectory + ";" + secondLibraryDirectory + ";" + firstLibraryDirectory
	server.loadProjects(params)
	scriptURI := workspace.PathToURI(script)
	for name, wantPath := range map[string]string{"Adjacent": adjacent, "First": first, "Second": second} {
		definition := server.definitionAt(scriptURI, name)
		if definition == nil || definition.URI != workspace.PathToURI(wantPath) {
			t.Errorf("%s definition = %#v, want %s", name, definition, wantPath)
		}
	}
	if definition := server.definitionAt(scriptURI, "Ignored"); definition != nil {
		t.Fatalf("non-library source was discovered: %#v", definition)
	}
	if definition := server.definitionAt(scriptURI, "Shared"); definition == nil || definition.URI != workspace.PathToURI(adjacent) {
		t.Fatalf("search order selected %#v, want adjacent library", definition)
	}
}

func TestSingleFileUsesUPLibPathEnvironment(t *testing.T) {
	root := t.TempDir()
	libraryDirectory := filepath.Join(root, "Libraries")
	if err := os.MkdirAll(libraryDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(root, "main.upscr")
	library := filepath.Join(libraryDirectory, "environment.uplib")
	if err := os.WriteFile(script, []byte("FromEnvironment()\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(library, []byte("proc FromEnvironment() {\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("UP_LIB_PATH", libraryDirectory)

	server := NewLsp(bytes.NewReader(nil), &bytes.Buffer{})
	rootURI := workspace.PathToURI(root)
	server.loadProjects(initializeParams{RootURI: &rootURI})
	definition := server.definitionAt(workspace.PathToURI(script), "FromEnvironment")
	if definition == nil || definition.URI != workspace.PathToURI(library) {
		t.Fatalf("environment library definition = %#v", definition)
	}
}

func TestConfiguredLibraryPathOverridesEnvironment(t *testing.T) {
	root := t.TempDir()
	configuredDirectory := filepath.Join(root, "Configured")
	environmentDirectory := filepath.Join(root, "Environment")
	for _, directory := range []string{configuredDirectory, environmentDirectory} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	script := filepath.Join(root, "main.upscr")
	configuredLibrary := filepath.Join(configuredDirectory, "configured.uplib")
	environmentLibrary := filepath.Join(environmentDirectory, "environment.uplib")
	for path, text := range map[string]string{
		script:             "Configured()\nEnvironment()\n",
		configuredLibrary:  "proc Configured() {\n}\n",
		environmentLibrary: "proc Environment() {\n}\n",
	} {
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("UP_LIB_PATH", environmentDirectory)

	server := NewLsp(bytes.NewReader(nil), &bytes.Buffer{})
	rootURI := workspace.PathToURI(root)
	params := initializeParams{RootURI: &rootURI}
	params.InitializationOptions.CMCLibraryPath = configuredDirectory
	server.loadProjects(params)
	scriptURI := workspace.PathToURI(script)
	if definition := server.definitionAt(scriptURI, "Configured"); definition == nil || definition.URI != workspace.PathToURI(configuredLibrary) {
		t.Fatalf("configured library definition = %#v", definition)
	}
	if definition := server.definitionAt(scriptURI, "Environment"); definition != nil {
		t.Fatalf("environment path was not overridden: %#v", definition)
	}
}

func TestUPLibPathDoesNotLeakIntoProjectScope(t *testing.T) {
	root := t.TempDir()
	projectDirectory := filepath.Join(root, "Project")
	libraryDirectory := filepath.Join(root, "ExternalLibraries")
	if err := os.MkdirAll(projectDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(libraryDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(projectDirectory, "main.upscr")
	externalLibrary := filepath.Join(libraryDirectory, "external.uplib")
	projectFile := filepath.Join(projectDirectory, "machine.upproj")
	if err := os.WriteFile(script, []byte("External()\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(externalLibrary, []byte("proc External() {\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectFile, []byte(`<pack><script ref="main.upscr" /></pack>`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("UP_LIB_PATH", libraryDirectory)

	server := NewLsp(bytes.NewReader(nil), &bytes.Buffer{})
	rootURI := workspace.PathToURI(root)
	server.loadProjects(initializeParams{RootURI: &rootURI})
	if definition := server.definitionAt(workspace.PathToURI(script), "External"); definition != nil {
		t.Fatalf("CMC Diff library leaked into .upproj scope: %#v", definition)
	}
}
