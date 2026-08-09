package lexer

import "github.com/lehmichael/cmc-lsp-go/internal/source"

// StringReplacement describes a $(...) replacement embedded in a quoted CMC
// string. The outer lexer deliberately keeps the complete string as one token;
// Tokens contains a second, position-adjusted tokenization of the replacement
// expression without the surrounding $( and ).
type StringReplacement struct {
	Range  source.SourceRange
	Tokens []Token
}

func StringReplacements(token Token) []StringReplacement {
	if token.Kind != LiteralString {
		return nil
	}
	runes := []rune(token.Lexeme)
	var result []StringReplacement
	for start := 0; start+1 < len(runes); start++ {
		if runes[start] != '$' || runes[start+1] != '(' {
			continue
		}
		depth := 1
		end := start + 2
		for ; end < len(runes); end++ {
			switch runes[end] {
			case '(':
				depth++
			case ')':
				depth--
				if depth == 0 {
					inner := string(runes[start+2 : end])
					tokens, _ := Tokenize(inner)
					shiftTokens(tokens, token.Range.Start.Line, token.Range.Start.Column+start+2)
					result = append(result, StringReplacement{
						Range: source.SourceRange{
							Start: source.SourceLocation{Line: token.Range.Start.Line, Column: token.Range.Start.Column + start},
							End:   source.SourceLocation{Line: token.Range.Start.Line, Column: token.Range.Start.Column + end + 1},
						},
						Tokens: tokens,
					})
					start = end
				}
			}
			if depth == 0 {
				break
			}
		}
	}
	return result
}

func shiftTokens(tokens []Token, line, column int) {
	for index := range tokens {
		tokens[index].Range.Start.Line += line
		tokens[index].Range.End.Line += line
		if tokens[index].Range.Start.Line == line {
			tokens[index].Range.Start.Column += column
		}
		if tokens[index].Range.End.Line == line {
			tokens[index].Range.End.Column += column
		}
	}
}
