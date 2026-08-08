// Package formatter implements deterministic formatting for CMC scripts.
package formatter

import (
	"strings"

	"github.com/lehmichael/cmc-lsp-go/internal/lexer"
)

type Options struct {
	TabSize       int
	InsertSpaces  bool
	CommentSpaces int
}

func DefaultOptions() Options {
	return Options{TabSize: 4, InsertSpaces: true, CommentSpaces: 2}
}

// Format normalizes indentation and intra-line whitespace while preserving
// comments, literals, blank lines and the spelling/case of language elements.
func Format(input string, options Options) string {
	if input == "" {
		return ""
	}
	if options.TabSize <= 0 {
		options.TabSize = 4
	}
	if options.CommentSpaces <= 0 {
		options.CommentSpaces = 2
	}

	tokens, _ := lexer.Tokenize(input)
	var lines []string
	var line []lexer.Token
	indent := 0

	flush := func() {
		if dedents(line) && indent > 0 {
			indent--
		}
		lines = append(lines, formatLine(line, indent, options))
		if indents(line) {
			indent++
		}
		line = nil
	}

	for _, token := range tokens {
		switch token.Kind {
		case lexer.EOF:
			if len(line) > 0 {
				flush()
			}
		case lexer.NewLine:
			flush()
		default:
			line = append(line, token)
		}
	}

	return strings.Join(lines, "\n") + "\n"
}

func formatLine(tokens []lexer.Token, indent int, options Options) string {
	if len(tokens) == 0 {
		return ""
	}

	var result strings.Builder
	if options.InsertSpaces {
		result.WriteString(strings.Repeat(" ", indent*options.TabSize))
	} else {
		result.WriteString(strings.Repeat("\t", indent))
	}

	for i, token := range tokens {
		if i > 0 {
			result.WriteString(separator(tokens, i, options))
		}
		result.WriteString(token.Lexeme)
	}
	return strings.TrimRight(result.String(), " \t")
}

func separator(tokens []lexer.Token, index int, options Options) string {
	previous := tokens[index-1]
	current := tokens[index]

	if current.Kind == lexer.Comment {
		return strings.Repeat(" ", options.CommentSpaces)
	}
	if previous.Kind == lexer.Comment {
		return ""
	}
	if current.Kind == lexer.Section && (previous.Kind == lexer.KeywordNamespaceNc ||
		previous.Kind == lexer.KeywordNamespacePs || previous.Kind == lexer.KeywordNamespaceBd) {
		return ""
	}
	if noSpaceBefore(current.Kind) || noSpaceAfter(previous.Kind) {
		return ""
	}
	if current.Kind == lexer.SymbolLeftParen {
		if previous.Kind == lexer.KeywordIf || previous.Kind == lexer.KeywordWhile ||
			previous.Kind == lexer.KeywordElseIf {
			return " "
		}
		return ""
	}
	if current.Kind == lexer.SymbolLeftBrace {
		return " "
	}
	if current.Kind == lexer.OperatorDelete || previous.Kind == lexer.OperatorDelete {
		return " "
	}
	if isOperator(current.Kind) {
		if isPrefix(tokens, index) {
			switch previous.Kind {
			case lexer.SymbolLeftParen, lexer.SymbolLeftBracket:
				return ""
			default:
				return " "
			}
		}
		return " "
	}
	if isOperator(previous.Kind) {
		if isPrefix(tokens, index-1) {
			return ""
		}
		return " "
	}
	if previous.Kind == lexer.SymbolComma || wordsNeedSpace(previous.Kind, current.Kind) {
		return " "
	}
	return ""
}

func noSpaceBefore(kind lexer.TokenKind) bool {
	switch kind {
	case lexer.SymbolRightParen, lexer.SymbolRightBracket, lexer.SymbolDot,
		lexer.SymbolComma, lexer.SymbolLeftBracket:
		return true
	default:
		return false
	}
}

func noSpaceAfter(kind lexer.TokenKind) bool {
	switch kind {
	case lexer.SymbolLeftParen, lexer.SymbolLeftBracket, lexer.SymbolDot,
		lexer.SymbolDollarParen:
		return true
	default:
		return false
	}
}

func wordsNeedSpace(left, right lexer.TokenKind) bool {
	return isWord(left) && (isWord(right) || right == lexer.LiteralString || right == lexer.LiteralNumberFormat)
}

func isWord(kind lexer.TokenKind) bool {
	switch kind {
	case lexer.LiteralIdentifier, lexer.LiteralBlockNumber, lexer.LiteralNumber,
		lexer.LiteralHex, lexer.LiteralNumberEx, lexer.LiteralString, lexer.LiteralNumberFormat,
		lexer.LiteralNull, lexer.LiteralTrue, lexer.LiteralFalse,
		lexer.KeywordIf, lexer.KeywordElse, lexer.KeywordElseIf, lexer.KeywordEndIf,
		lexer.KeywordWhile, lexer.KeywordEndWhile, lexer.KeywordProc, lexer.KeywordFunc,
		lexer.KeywordReturn, lexer.KeywordNamespaceNc, lexer.KeywordNamespacePs,
		lexer.KeywordNamespaceBd, lexer.KeywordNamespaceChan, lexer.PreprocessorInclude,
		lexer.PreprocessorUnknown, lexer.Section:
		return true
	default:
		return false
	}
}

func isOperator(kind lexer.TokenKind) bool {
	switch kind {
	case lexer.OperatorAssign, lexer.OperatorAssignRaw, lexer.OperatorAssignIfBlank,
		lexer.OperatorAddAssign, lexer.OperatorSubtractAssign, lexer.OperatorMultiplyAssign,
		lexer.OperatorDivideAssign, lexer.OperatorOrAssign, lexer.OperatorAndAssign,
		lexer.OperatorEqual, lexer.OperatorUnequal, lexer.OperatorLessThan,
		lexer.OperatorLessThanEqual, lexer.OperatorGreaterThan, lexer.OperatorGreaterThanEqual,
		lexer.OperatorLogAnd, lexer.OperatorLogOr, lexer.OperatorStringConcat,
		lexer.OperatorAdd, lexer.OperatorSubtract, lexer.OperatorMultiply,
		lexer.OperatorDivide, lexer.OperatorAnd, lexer.OperatorOr, lexer.OperatorNegate:
		return true
	default:
		return false
	}
}

func isPrefix(tokens []lexer.Token, index int) bool {
	kind := tokens[index].Kind
	if kind != lexer.OperatorAdd && kind != lexer.OperatorSubtract && kind != lexer.OperatorNegate {
		return false
	}
	if index == 0 {
		return true
	}
	previous := tokens[index-1].Kind
	return isOperator(previous) || previous == lexer.SymbolLeftParen ||
		previous == lexer.SymbolLeftBracket || previous == lexer.SymbolComma ||
		previous == lexer.KeywordIf || previous == lexer.KeywordWhile ||
		previous == lexer.KeywordElseIf
}

func firstCodeToken(tokens []lexer.Token) lexer.TokenKind {
	for _, token := range tokens {
		if token.Kind != lexer.Comment {
			return token.Kind
		}
	}
	return lexer.Unknown
}

func dedents(tokens []lexer.Token) bool {
	switch firstCodeToken(tokens) {
	case lexer.KeywordElse, lexer.KeywordElseIf, lexer.KeywordEndIf,
		lexer.KeywordEndWhile, lexer.SymbolRightBrace:
		return true
	default:
		return false
	}
}

func indents(tokens []lexer.Token) bool {
	switch firstCodeToken(tokens) {
	case lexer.KeywordIf, lexer.KeywordWhile, lexer.KeywordElse, lexer.KeywordElseIf:
		return true
	case lexer.KeywordFunc, lexer.KeywordProc:
		for _, token := range tokens {
			if token.Kind == lexer.SymbolLeftBrace {
				return true
			}
		}
	}
	return false
}
