package formatter

import (
	"strings"
	"testing"
)

func TestFormat(t *testing.T) {
	input := "; heading\r\nIF(up.selection==1) ; branch\r\nLog(\"one\")\r\nELSIF (up.selection==2)\r\n$MA_MAX_AX_VELO[AX1]=up.value+100\r\nELSE\r\nup.raw:=NC[C1].$MC_CHAN_NAME\r\nENDIF"
	want := "; heading\nIF (up.selection == 1)  ; branch\n    Log(\"one\")\nELSIF (up.selection == 2)\n    $MA_MAX_AX_VELO[AX1] = up.value + 100\nELSE\n    up.raw := NC[C1].$MC_CHAN_NAME\nENDIF\n"
	if got := Format(input, DefaultOptions()); got != want {
		t.Fatalf("Format mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatLibraryAndReplacement(t *testing.T) {
	input := "func MyFunction($,$){\nIf !Up.ok\nUp.value=-1\nEndIf\nReturn(-1)\n}\n[B3_S$(Up.slave)_PS$(Up.doNr)]\np107[$(Up.i)]?=254\n"
	want := "func MyFunction($, $) {\n    If !Up.ok\n        Up.value = -1\n    EndIf\n    Return(-1)\n}\n[B3_S$(Up.slave)_PS$(Up.doNr)]\np107[$(Up.i)] ?= 254\n"
	if got := Format(input, DefaultOptions()); got != want {
		t.Fatalf("Format mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
	if got := Format(want, DefaultOptions()); got != want {
		t.Fatalf("Format is not idempotent\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestConfigurableIndentAndComments(t *testing.T) {
	options := Options{TabSize: 8, InsertSpaces: false, CommentSpaces: 3}
	input := "If true ; note\nUp.x=1\nEndIf\n"
	want := "If true   ; note\n\tUp.x = 1\nEndIf\n"
	if got := Format(input, options); got != want {
		t.Fatalf("Format mismatch\nwant:\n%q\ngot:\n%q", want, got)
	}
}

func TestAlignConsecutiveAssignmentsAndTrailingComments(t *testing.T) {
	input := `Up.x=1 ; one
Up.longName:=22 ; two
Up.z?=333 ; three

Up.a=1 ; separate
; group break
Up.bb=22 ; separate too
If true
Up.short=1 ; nested
Up.considerablyLonger=22 ; nested two
EndIf
`
	got := Format(input, DefaultOptions())
	lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	assertSameColumns(t, lines[:3], assignmentColumn)
	assertSameColumns(t, lines[:3], commentColumn)
	if lines[0] != "Up.x        = 1     ; one" ||
		lines[1] != "Up.longName := 22   ; two" ||
		lines[2] != "Up.z        ?= 333  ; three" {
		t.Fatalf("top alignment mismatch:\n%s", strings.Join(lines[:3], "\n"))
	}
	if lines[4] != "Up.a = 1  ; separate" || lines[6] != "Up.bb = 22  ; separate too" {
		t.Fatalf("group boundaries were ignored:\n%s", got)
	}
	assertSameColumns(t, lines[8:10], assignmentColumn)
	assertSameColumns(t, lines[8:10], commentColumn)
	if second := Format(got, DefaultOptions()); second != got {
		t.Fatalf("aligned formatting is not idempotent\nfirst:\n%s\nsecond:\n%s", got, second)
	}
}

func TestAlignmentCanBeDisabled(t *testing.T) {
	options := DefaultOptions()
	options.AlignConsecutiveAssignments = false
	options.AlignTrailingComments = false
	input := "Up.x=1 ; one\nUp.longName=22 ; two\n"
	want := "Up.x = 1  ; one\nUp.longName = 22  ; two\n"
	if got := Format(input, options); got != want {
		t.Fatalf("Format mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestAlignmentUsesVisualColumnsWithTabsAndUnicode(t *testing.T) {
	options := DefaultOptions()
	options.InsertSpaces = false
	input := "If true\nÄ.x=1 ; eins\nÄ.länger=22 ; zwei\nEndIf\n"
	lines := strings.Split(strings.TrimSuffix(Format(input, options), "\n"), "\n")
	assertSameColumns(t, lines[1:3], assignmentColumn)
	assertSameColumns(t, lines[1:3], commentColumn)
	if !strings.HasPrefix(lines[1], "\t") || !strings.HasPrefix(lines[2], "\t") {
		t.Fatalf("tab indentation was not preserved: %#v", lines)
	}
}

func assertSameColumns(t *testing.T, lines []string, column func(string) int) {
	t.Helper()
	want := column(lines[0])
	for _, line := range lines[1:] {
		if got := column(line); got != want {
			t.Fatalf("columns differ: %d and %d in %#v", want, got, lines)
		}
	}
}

func assignmentColumn(line string) int {
	for _, operator := range []string{" := ", " ?= ", " += ", " -= ", " *= ", " /= ", " |= ", " &= ", " = "} {
		if index := strings.Index(line, operator); index >= 0 {
			return visualWidth(line[:index+1], 4)
		}
	}
	return -1
}

func commentColumn(line string) int {
	index := strings.Index(line, ";")
	if index < 0 {
		return -1
	}
	return visualWidth(line[:index], 4)
}
