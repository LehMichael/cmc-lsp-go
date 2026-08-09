package database

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestLoadLookupAndLocale(t *testing.T) {
	root := t.TempDir()
	english := filepath.Join(root, "en")
	if err := os.Mkdir(english, 0o755); err != nil {
		t.Fatal(err)
	}
	germanXML := `<?xml version="1.0" encoding="windows-1252"?><info><parameter number="10000" type="STRING" dim="1"><name>AXCONF_MACHAX_NAME_TAB</name><brief>Maschinenachsname</brief><description>Deutsche Beschreibung.</description></parameter></info>`
	englishXML := `<?xml version="1.0" encoding="windows-1252"?><info><parameter number="10000" type="STRING" dim="1"><name>AXCONF_MACHAX_NAME_TAB</name><brief>Machine axis name</brief><description>English description.</description></parameter></info>`
	writeTestFile(t, filepath.Join(root, "mdnck.mdat"), germanXML)
	writeTestFile(t, filepath.Join(english, "mdnck.mdat"), englishXML)
	writeTestFile(t, filepath.Join(english, "cmdnck.mdat"), `<?xml version="1.0"?><info><parameter number="51000" type="BYTE" dim="0"><name>DISPLAY_TEST</name><brief>Cycle machine data</brief></parameter></info>`)
	writeTestFile(t, filepath.Join(english, "mdchan.mdat"), `<?xml version="1.0"?><info><parameter number="24000" type="BYTE" dim="0"><name>TRAFO_TYPE_1</name><brief>First transformation</brief></parameter><parameter number="24400" type="BYTE" dim="0"><name>TRAFO_TYPE_2</name><brief>Second transformation</brief></parameter></info>`)
	writeTestFile(t, filepath.Join(english, "do001.para"), `<?xml version="1.0"?><info comment="DEVICE"><parameter number="10" type="Int16" dim="0"><name>Test</name><brief>Writable drive value</brief></parameter><parameter number="2" type="Int16" dim="0" readonly="true"><name>Status</name><brief>Read-only drive value</brief></parameter></info>`)
	writeTestFile(t, filepath.Join(english, "vars.svar"), `<?xml version="1.0"?><info><parameter type="DOUBLE" dim="1"><name>$TC_TEST</name><brief>System value</brief></parameter></info>`)

	englishCatalog, err := Load(root, "en-US")
	if err != nil {
		t.Fatal(err)
	}
	if got := englishCatalog.Lookup("$mn_axconf_machax_name_tab"); len(got) != 1 || got[0].Brief != "Machine axis name" {
		t.Fatalf("English machine data = %#v", got)
	}
	if got := englishCatalog.Lookup("P0010"); len(got) != 1 || got[0].Identifier != "p10" {
		t.Fatalf("drive parameter = %#v", got)
	}
	if got := englishCatalog.Lookup("r2"); len(got) != 1 || !got[0].ReadOnly {
		t.Fatalf("read-only drive parameter = %#v", got)
	}
	if got := englishCatalog.Lookup("$MNS_DISPLAY_TEST"); len(got) != 1 || got[0].Number != "51000" {
		t.Fatalf("cycle machine data = %#v", got)
	}
	if got := englishCatalog.Lookup("$MC_TRAFO_TYPE_"); len(got) != 2 {
		t.Fatalf("dynamic machine-data family = %#v", got)
	}
	if hover, ok := englishCatalog.Hover("$TC_TEST"); !ok || !strings.Contains(hover, "System value") {
		t.Fatalf("system-variable hover = %q, %v", hover, ok)
	}

	germanCatalog, err := Load(root, "de-AT")
	if err != nil {
		t.Fatal(err)
	}
	if got := germanCatalog.Lookup("$MN_AXCONF_MACHAX_NAME_TAB"); len(got) != 1 || got[0].Brief != "Maschinenachsname" {
		t.Fatalf("German machine data = %#v", got)
	}
}

func TestDatabaseCandidatesInAncestors(t *testing.T) {
	root := t.TempDir()
	binaryDirectory := filepath.Join(root, "cmc-lsp-go", "bin")
	candidates := databaseCandidatesInAncestors(binaryDirectory)
	want := filepath.Join(root, "cmc-lsp-go", "DataBase")
	if !slices.Contains(candidates, want) {
		t.Fatalf("candidates = %#v, missing %q", candidates, want)
	}
}

func TestLocate(t *testing.T) {
	root := t.TempDir()
	databaseDirectory := filepath.Join(root, "DataBase")
	workspace := filepath.Join(root, "projects", "machine")
	if err := os.MkdirAll(filepath.Join(databaseDirectory, "en"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(databaseDirectory, "en", "values.svar"), `<?xml version="1.0"?><info/>`)
	if got := Locate([]string{workspace}, ""); got != databaseDirectory {
		t.Fatalf("Locate() = %q, want %q", got, databaseDirectory)
	}
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
