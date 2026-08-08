package formatter

import "testing"

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
