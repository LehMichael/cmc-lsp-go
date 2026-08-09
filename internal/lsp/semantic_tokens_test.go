package lsp

import (
	"slices"
	"testing"
)

func TestSemanticTokens(t *testing.T) {
	text := "; heading\nfunc Measure($) {\n    Up.result = Measure(0x2A)\n}\n"
	data := semanticTokensFor(text)
	decoded := decodeSemanticTokens(data)

	want := []decodedSemanticToken{
		{0, 0, 9, "comment", 0},
		{1, 0, 4, "keyword", 0},
		{1, 5, 7, "function", semanticDeclaration},
		{1, 13, 1, "parameter", 0},
		{2, 4, 2, "namespace", 0},
		{2, 7, 6, "property", 0},
		{2, 14, 1, "operator", 0},
		{2, 16, 7, "function", 0},
		{2, 24, 4, "number", 0},
	}
	if !slices.Equal(decoded, want) {
		t.Fatalf("semantic tokens\nwant: %#v\n got: %#v", want, decoded)
	}
}

func TestSemanticTokensUseUTF16Columns(t *testing.T) {
	decoded := decodeSemanticTokens(semanticTokensFor("Up.ä = \"😀\"\n"))
	want := []decodedSemanticToken{
		{0, 0, 2, "namespace", 0},
		{0, 3, 1, "property", 0},
		{0, 5, 1, "operator", 0},
		{0, 7, 4, "string", 0},
	}
	if !slices.Equal(decoded, want) {
		t.Fatalf("semantic tokens\nwant: %#v\n got: %#v", want, decoded)
	}
}

func TestSemanticTokensTreatLeadingNNumberAsAnnotation(t *testing.T) {
	decoded := decodeSemanticTokens(semanticTokensFor("  N20000 Up.x = 1\nUp.N20000 = 2\n"))
	want := []decodedSemanticToken{
		{0, 2, 6, "comment", 0},
		{0, 9, 2, "namespace", 0},
		{0, 12, 1, "property", 0},
		{0, 14, 1, "operator", 0},
		{0, 16, 1, "number", 0},
		{1, 0, 2, "namespace", 0},
		{1, 3, 6, "property", 0},
		{1, 10, 1, "operator", 0},
		{1, 12, 1, "number", 0},
	}
	if !slices.Equal(decoded, want) {
		t.Fatalf("semantic tokens\nwant: %#v\n got: %#v", want, decoded)
	}
}

func TestSemanticTokensHighlightReplacementsInsideStrings(t *testing.T) {
	decoded := decodeSemanticTokens(semanticTokensFor("Up.text = \"x$(Up.path)y\"\n"))
	want := []decodedSemanticToken{
		{0, 0, 2, "namespace", 0},
		{0, 3, 4, "property", 0},
		{0, 8, 1, "operator", 0},
		{0, 10, 2, "string", 0},
		{0, 14, 2, "namespace", 0},
		{0, 17, 4, "property", 0},
		{0, 22, 2, "string", 0},
	}
	if !slices.Equal(decoded, want) {
		t.Fatalf("semantic tokens\nwant: %#v\n got: %#v", want, decoded)
	}
}

type decodedSemanticToken struct {
	line, start, length int
	tokenType           string
	modifiers           uint32
}

func decodeSemanticTokens(data []uint32) []decodedSemanticToken {
	var result []decodedSemanticToken
	line := 0
	start := 0
	for index := 0; index+4 < len(data); index += 5 {
		line += int(data[index])
		if data[index] == 0 {
			start += int(data[index+1])
		} else {
			start = int(data[index+1])
		}
		result = append(result, decodedSemanticToken{
			line: line, start: start, length: int(data[index+2]),
			tokenType: semanticTokenTypes[data[index+3]], modifiers: data[index+4],
		})
	}
	return result
}
