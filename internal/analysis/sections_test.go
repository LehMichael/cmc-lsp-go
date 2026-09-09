package analysis

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lehmichael/cmc-lsp-go/internal/diag"
	"github.com/lehmichael/cmc-lsp-go/internal/lexer"
	"github.com/lehmichael/cmc-lsp-go/internal/parser"
	"github.com/lehmichael/cmc-lsp-go/internal/workspace"
)

func TestSectionAccessRequiresMatchingActiveSection(t *testing.T) {
	diagnostics := validateSections(t, `$MC_CHAN_NAME = "missing NC"
p105 = 1
$MM_SHOW_TOOLTIP = 1
NC[C1]
$MC_CHAN_NAME = "valid"
p105 = 2
PS[B3_S3_PS3]
p105 = 3
$MC_CHAN_NAME = "wrong section"
BD[SL]
$MM_SHOW_TOOLTIP = 0
r105 = 4
`)
	wantKinds := []diag.DiagnosticKind{
		diag.NCDataSectionRequired,
		diag.DriveDataSectionRequired,
		diag.DisplayDataSectionRequired,
		diag.DriveDataSectionRequired,
		diag.NCDataSectionRequired,
		diag.DriveDataSectionRequired,
	}
	wantLines := []int{0, 1, 2, 5, 8, 11}
	assertDiagnostics(t, diagnostics, wantKinds, wantLines)
}

func TestFullyQualifiedAreaAccessIsReadOnly(t *testing.T) {
	diagnostics := validateSections(t, `Up.nc = NC[C1].$MC_CHAN_NAME
Up.drive = PS[B3_S3_PS3].P105
Up.display = BD[SL].$MM_SHOW_TOOLTIP
Up.badNC = PS[B3_S3_PS3].$MC_CHAN_NAME
Up.badDrive = NC[C1].P105
Up.badDisplay = NC[C1].$MM_SHOW_TOOLTIP
NC[C1].$MC_CHAN_NAME = "write"
PS[B3_S3_PS3].P105 = 1
BD[SL].$MM_SHOW_TOOLTIP~
NC[C1].$(Up.dynamicName) = 1
`)
	wantKinds := []diag.DiagnosticKind{
		diag.NCDataSectionRequired,
		diag.DriveDataSectionRequired,
		diag.DisplayDataSectionRequired,
		diag.FullyQualifiedIdentifierWrite,
		diag.FullyQualifiedIdentifierWrite,
		diag.FullyQualifiedIdentifierWrite,
		diag.FullyQualifiedIdentifierWrite,
	}
	wantLines := []int{3, 4, 5, 6, 7, 8, 9}
	assertDiagnostics(t, diagnostics, wantKinds, wantLines)
}

func TestSectionAccessThroughControlFlow(t *testing.T) {
	diagnostics := validateSections(t, `NC[C1]
If Up.useDrive
    p105 = 1
Else
    PS[B3_S3_PS3]
    p105 = 2
EndIf
$MC_CHAN_NAME = "possibly NC"
p105 = 3
PS[B3_S3_PS3]
While Up.more
    $MC_CHAN_NAME = "wrong on first iteration"
    NC[C1]
EndWhile
p105 = 4
func UsesCallerSection() {
    $MC_CHAN_NAME = "caller-dependent"
    PS[B3_S3_PS3]
    $MC_CHAN_NAME = "definitely wrong"
}
`)
	wantKinds := []diag.DiagnosticKind{
		diag.DriveDataSectionRequired,
		diag.NCDataSectionRequired,
		diag.NCDataSectionRequired,
	}
	wantLines := []int{2, 11, 18}
	assertDiagnostics(t, diagnostics, wantKinds, wantLines)
}

func TestDynamicSectionKeepsItsArea(t *testing.T) {
	diagnostics := validateSections(t, `PS[$(Up.drivePath)]
p105 = 1
NC[C$(Up.channel)]
$MCS_NAME = "channel"
BD[$(Up.displaySection)]
$MM_SHOW_TOOLTIP = 0
[$(Up.unknownSection)]
$MC_CHAN_NAME = "runtime-dependent"
p105 = 2
`)
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
}

func TestSimpleSectionSelectorsChooseTheirArea(t *testing.T) {
	diagnostics := validateSections(t, `[C1]
$MC_CHAN_NAME = "channel"
[B3_S3_PS3]
p105 = 1
[SL]
$MM_SHOW_TOOLTIP = 0
`)
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
}

func TestIncludedScriptReturnsItsActiveSection(t *testing.T) {
	directory := t.TempDir()
	root := filepath.Join(directory, "root.upscr")
	include := filepath.Join(directory, "drive.upscr")
	if err := os.WriteFile(include, []byte("PS[B3_S3_PS3]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	input := "#include \"drive.upscr\"\np105 = 1\n$MC_CHAN_NAME = \"wrong after include\"\n"
	if err := os.WriteFile(root, []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}
	ast := parseSections(t, input)
	diagnostics := ValidateSectionAccesses(root, ast, workspace.NewOverlay())
	assertDiagnostics(t, diagnostics, []diag.DiagnosticKind{diag.NCDataSectionRequired}, []int{2})
}

func validateSections(t *testing.T, input string) []diag.Diagnostic {
	t.Helper()
	return ValidateSectionAccesses("test.upscr", parseSections(t, input), nil)
}

func parseSections(t *testing.T, input string) parser.Ast {
	t.Helper()
	tokens, diagnostics := lexer.Tokenize(input)
	ast, diagnostics := parser.Parse(tokens, diagnostics)
	if len(diagnostics) != 0 {
		t.Fatalf("parse diagnostics = %#v", diagnostics)
	}
	return ast
}

func assertDiagnostics(t *testing.T, diagnostics []diag.Diagnostic, wantKinds []diag.DiagnosticKind, wantLines []int) {
	t.Helper()
	if len(diagnostics) != len(wantKinds) {
		t.Fatalf("diagnostics = %#v, want %d", diagnostics, len(wantKinds))
	}
	for index, diagnostic := range diagnostics {
		if diagnostic.Kind != wantKinds[index] || diagnostic.Range.Start.Line != wantLines[index] {
			t.Errorf("diagnostic %d = %#v, want kind %v on line %d", index, diagnostic, wantKinds[index], wantLines[index])
		}
	}
}
