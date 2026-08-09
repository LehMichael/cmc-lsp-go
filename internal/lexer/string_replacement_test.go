package lexer

import "testing"

func TestStringReplacements(t *testing.T) {
	tokens, diagnostics := Tokenize("\nUp.text = \"a$(Up.x)b$(Tmp.name)c\"\n")
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
	var stringToken Token
	for _, token := range tokens {
		if token.Kind == LiteralString {
			stringToken = token
			break
		}
	}
	replacements := StringReplacements(stringToken)
	if len(replacements) != 2 {
		t.Fatalf("replacements = %#v", replacements)
	}
	if replacements[0].Range.Start.Line != 1 || replacements[0].Range.Start.Column != 12 || replacements[0].Range.End.Column != 19 {
		t.Fatalf("first replacement range = %#v", replacements[0].Range)
	}
	want := []string{"Up", ".", "x"}
	for index, lexeme := range want {
		if replacements[0].Tokens[index].Lexeme != lexeme {
			t.Fatalf("first replacement tokens = %#v", replacements[0].Tokens)
		}
	}
	if replacements[1].Tokens[0].Range.Start.Column != 22 || replacements[1].Tokens[2].Lexeme != "name" {
		t.Fatalf("second replacement tokens = %#v", replacements[1].Tokens)
	}
}

func TestStringReplacementsIgnoreIncompleteOperator(t *testing.T) {
	token := Token{Kind: LiteralString, Lexeme: `"$(Up.path"`}
	if replacements := StringReplacements(token); len(replacements) != 0 {
		t.Fatalf("replacements = %#v", replacements)
	}
}
