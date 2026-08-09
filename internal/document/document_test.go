package document

import (
	"strings"
	"testing"

	"github.com/lehmichael/cmc-lsp-go/internal/formatter"
	"github.com/lehmichael/cmc-lsp-go/internal/lexer"
	"github.com/lehmichael/cmc-lsp-go/internal/parser"
)

func TestCMCTextExtractsUpactScript(t *testing.T) {
	input := "<?xml version=\"1.0\"?>\n<action>\n<script>If Up.a == true &amp;&amp; Up.$Pack.NCU == true\nEndIf\n</script>\n</action>\n"
	masked := CMCText("deploy.upact", input)
	if strings.Count(masked, "\n") != strings.Count(input, "\n") || !strings.Contains(masked, "If Up.a == true &&") {
		t.Fatalf("masked document = %q", masked)
	}
	tokens, diagnostics := lexer.Tokenize(masked)
	_, diagnostics = parser.Parse(tokens, diagnostics)
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
}

func TestFormatPreservesUpactXML(t *testing.T) {
	input := "<?xml version=\"1.0\"?>\n<action type=\"copy\"><script>If Up.a==true &amp;&amp; Up.b==true\nUp.x=1\nEndIf\n</script><copy><obj name=\"NCU\" /></copy></action>\n"
	formatted := Format("deploy.upact", input, formatter.DefaultOptions())
	if !strings.Contains(formatted, "<action type=\"copy\"><script>") || !strings.Contains(formatted, "</script><copy><obj name=\"NCU\" /></copy></action>") {
		t.Fatalf("XML changed: %q", formatted)
	}
	if !strings.Contains(formatted, "If Up.a == true &amp;&amp; Up.b == true\n    Up.x = 1\nEndIf") {
		t.Fatalf("script not formatted: %q", formatted)
	}
}

func TestSemanticTextMasksEntities(t *testing.T) {
	input := "<script>If Up.a &amp;&amp; Up.b\nEndIf</script>"
	semantic := SemanticText("deploy.upact", input)
	if strings.Contains(semantic, "amp") || !strings.Contains(semantic, "Up.a") || !strings.Contains(semantic, "Up.b") {
		t.Fatalf("semantic document = %q", semantic)
	}
}
