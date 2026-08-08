package parser

import (
	"testing"

	"github.com/lehmichael/cmc-lsp-go/internal/lexer"
)

func TestManualLanguageExamples(t *testing.T) {
	tests := map[string]string{
		"control flow and indices": `[B3_S3_PS99]
Up.i = 0
Up.CU_Hat_Servo = false
While Up.i < 6 && !Up.CU_Hat_Servo
  If p978[$(Up.i)] != 255
    Up.CU_Hat_Servo = true
  ElsIf p978[0] == 254
    Up.i += 1
  ElIf p978[0] == 253
    Up.i -= 1
  Else
    Up.i = 6
  EndIf
EndWhile
`,
		"qualified data and sections": `[C1]
$MC_CHAN_NAME = "Channel 1"
Up.NAME_CHAN_3 = NC[C3].$MC_CHAN_NAME
PS[B3_S3_PS1].p105 = 1
BD[SL].$MM_SHOW_TOOLTIP = 1
CHANDATA(3)
[$(Up.doX.psPath)]
p107[$(Up.i)] ?= 254
`,
		"literals and operators": `N20000 $MC_CHAN_NAME="Machine"
$MN_TOOL_MANAGEMENT_MASK |= 'B10000'
Up.Mask = 'HFF'
Up.Link = 1:105.0
Up.DynamicLink = $(Up.doNr):4105.0
Up.Small = 1.2EX-3
Up.DriveSmall = 1.05E-02
Up.Text = "line 1" << '\r' << '\n' << "line 2"
Up.Raw := $MA_MAX_AX_VELO[AX1]
`,
		"library": `;Description: example
;Arg1: first value
func MyFunction($, $) {
  Up.Arg1 = Up.$1
  Up.Arg2 = Up.$2
  If Up.Arg1 == true
    Return(0)
  Else
    Return(-1)
  EndIf
}

proc MyProcedure() {
  Msg("done")
}
`,
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			tokens, diagnostics := lexer.Tokenize(input)
			_, diagnostics = Parse(tokens, diagnostics)
			if len(diagnostics) != 0 {
				t.Fatalf("expected no diagnostics, got %#v", diagnostics)
			}
		})
	}
}

func TestMalformedInputRecovers(t *testing.T) {
	inputs := []string{
		"@\n",
		"#include",
		"If (true\n",
		"While\n",
		"foo(,\n",
		"[B]\n",
		"PS(00000000000000",
		"func () {\n",
		"$(Up.missing\n",
	}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			tokens, diagnostics := lexer.Tokenize(input)
			_, diagnostics = Parse(tokens, diagnostics)
			if len(diagnostics) == 0 {
				t.Fatal("malformed input produced no diagnostic")
			}
		})
	}
}

func FuzzParserDoesNotPanic(f *testing.F) {
	for _, seed := range []string{
		"", "@", "#include", "If true\nEndIf\n", "func F($) {\nReturn(1)\n}\n",
		"PS[B3_S3_PS1].p105 = 'HFF'", "[$(Up.path)]\np1[$(Up.i)] = 2\n",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		tokens, diagnostics := lexer.Tokenize(input)
		_, _ = Parse(tokens, diagnostics)
	})
}
