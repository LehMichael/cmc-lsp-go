package lexer

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/lehmichael/cmc-lsp-go/internal/diag"
	"github.com/lehmichael/cmc-lsp-go/internal/source"
)

const eof rune = -1

type Lexer struct {
	input         []rune
	pos           int
	currentLine   int
	currentColumn int
	lastTokenKind TokenKind
	diagnostics   []diag.Diagnostic
}

func NewLexer(input string) Lexer {
	return Lexer{
		input:         []rune(input),
		lastTokenKind: NewLine,
	}
}

func Tokenize(input string) ([]Token, []diag.Diagnostic) {
	l := NewLexer(input)

	var tokens []Token

	for {
		t := l.NextToken()
		tokens = append(tokens, t)
		if t.Kind == EOF {
			break
		}
	}

	return tokens, l.diagnostics
}

func (l *Lexer) NextToken() Token {
	whitespaceStart := l.pos

	for l.pos < len(l.input) && unicode.IsSpace(l.getCurrentChar()) && l.getCurrentChar() != '\n' {
		l.pos++
		l.currentColumn++
	}

	leadingWhitespace := string(l.input[whitespaceStart:l.pos])

	// fmt.Printf("pos: %v, len: %v\n", l.pos, len(l.input))

	if l.pos >= len(l.input) {
		return Token{
			Kind:              EOF,
			Lexeme:            "",
			LeadingWhitespace: leadingWhitespace,
			Range: source.SourceRange{
				Start: source.SourceLocation{Line: l.currentLine, Column: l.currentColumn},
				End:   source.SourceLocation{Line: l.currentLine, Column: l.currentColumn},
			},
		}
	}

	nextChar := l.getNextChar()
	c := l.getCurrentChar()
	var token Token
	switch {
	case c == ';':
		token = l.createTokeWhile(
			Comment,
			leadingWhitespace,
			func(c rune) bool { return c != '\n' && c != '\r' },
		)
	case c == '\n':
		token = l.createTokenCount(NewLine, leadingWhitespace, 1)
		l.currentLine++
		l.currentColumn = 0
	case c == '=' && nextChar == '=':
		token = l.createTokenCount(OperatorEqual, leadingWhitespace, 2)
	case c == '=':
		token = l.createTokenCount(OperatorAssign, leadingWhitespace, 1)
	case c == ':' && nextChar == '=':
		token = l.createTokenCount(OperatorAssignRaw, leadingWhitespace, 2)
	case c == '!' && nextChar == '=':
		token = l.createTokenCount(OperatorUnequal, leadingWhitespace, 2)
	case c == '!':
		token = l.createTokenCount(OperatorNegate, leadingWhitespace, 1)
	case c == '<' && nextChar == '=':
		token = l.createTokenCount(OperatorLessThanEqual, leadingWhitespace, 2)
	case c == '<' && nextChar == '<':
		token = l.createTokenCount(OperatorStringConcat, leadingWhitespace, 2)
	case c == '<':
		token = l.createTokenCount(OperatorLessThan, leadingWhitespace, 1)
	case c == '>' && nextChar == '=':
		token = l.createTokenCount(OperatorGreaterThanEqual, leadingWhitespace, 2)
	case c == '>':
		token = l.createTokenCount(OperatorGreaterThan, leadingWhitespace, 1)
	case c == '[' && (l.lastTokenKind == NewLine || l.lastTokenKind == KeywordNamespaceNc || l.lastTokenKind == KeywordNamespacePs || l.lastTokenKind == KeywordNamespaceBd):
		token = l.createDelimitedToken(
			Section,
			diag.SectionUnterminated,
			leadingWhitespace,
			']',
		)
	case c == '[':
		token = l.createTokenCount(SymbolLeftBracket, leadingWhitespace, 1)
	case c == ']':
		token = l.createTokenCount(SymbolRightBracket, leadingWhitespace, 1)
	case c == '(' && l.lastTokenKind == KeywordNamespaceChan:
		token = l.createDelimitedToken(
			Section,
			diag.SectionUnterminated,
			leadingWhitespace,
			')',
		)
	case c == '(':
		token = l.createTokenCount(SymbolLeftParen, leadingWhitespace, 1)
	case c == ')':
		token = l.createTokenCount(SymbolRightParen, leadingWhitespace, 1)
	case c == '{':
		token = l.createTokenCount(SymbolLeftBrace, leadingWhitespace, 1)
	case c == '}':
		token = l.createTokenCount(SymbolRightBrace, leadingWhitespace, 1)
	case c == '.':
		token = l.createTokenCount(SymbolDot, leadingWhitespace, 1)
	case c == ',':
		token = l.createTokenCount(SymbolComma, leadingWhitespace, 1)
	case c == '~':
		token = l.createTokenCount(OperatorDelete, leadingWhitespace, 1)
	case c == '+' && nextChar == '=':
		token = l.createTokenCount(OperatorAddAssign, leadingWhitespace, 2)
	case c == '+':
		token = l.createTokenCount(OperatorAdd, leadingWhitespace, 1)
	case c == '-' && nextChar == '=':
		token = l.createTokenCount(OperatorSubtractAssign, leadingWhitespace, 2)
	case c == '-':
		token = l.createTokenCount(OperatorSubtract, leadingWhitespace, 1)
	case c == '*' && nextChar == '=':
		token = l.createTokenCount(OperatorMultiplyAssign, leadingWhitespace, 2)
	case c == '*':
		token = l.createTokenCount(OperatorMultiply, leadingWhitespace, 1)
	case c == '/' && nextChar == '=':
		token = l.createTokenCount(OperatorDivideAssign, leadingWhitespace, 2)
	case c == '/':
		token = l.createTokenCount(OperatorDivide, leadingWhitespace, 1)
	case c == '|' && nextChar == '=':
		token = l.createTokenCount(OperatorOrAssign, leadingWhitespace, 2)
	case c == '|' && nextChar == '|':
		token = l.createTokenCount(OperatorLogOr, leadingWhitespace, 2)
	case c == '|':
		token = l.createTokenCount(OperatorOr, leadingWhitespace, 1)
	case c == '&' && nextChar == '=':
		token = l.createTokenCount(OperatorAndAssign, leadingWhitespace, 2)
	case c == '&' && nextChar == '&':
		token = l.createTokenCount(OperatorLogAnd, leadingWhitespace, 2)
	case c == '&':
		token = l.createTokenCount(OperatorAnd, leadingWhitespace, 1)
	case c == '"':
		token = l.createStringToken(leadingWhitespace)
	case c == '\'':
		token = l.createDelimitedToken(
			LiteralNumberFormat, diag.NumberFormatUnterminated, leadingWhitespace, '\'',
		)
	case c == '?' && nextChar == '=':
		token = l.createTokenCount(OperatorAssignIfBlank, leadingWhitespace, 2)
	case c == '$' && (unicode.IsLetter(nextChar) || unicode.IsNumber(nextChar) || nextChar == '_'):
		token = l.createTokeWhile(
			LiteralIdentifier,
			leadingWhitespace,
			func(r rune) bool { return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r) },
		)
	case c == '$' && nextChar == '(':
		token = l.createTokenCount(SymbolDollarParen, leadingWhitespace, 2)
	case c == '$':
		token = l.createTokenCount(SymbolDollar, leadingWhitespace, 1)
	case c == '0' && (nextChar == 'x' || nextChar == 'X'):
		token = l.createHexToken(leadingWhitespace)
	case unicode.IsNumber(c):
		token = l.createTokeWhile(
			LiteralNumber,
			leadingWhitespace,
			func(r rune) bool { return r == '.' || r == ':' || unicode.IsNumber(r) },
		)
	case c == '_' || unicode.IsLetter(c):
		token = l.createIdentifierToken(leadingWhitespace)
	case c == '#':
		token = l.createPreprocessorToken(leadingWhitespace)
	default:
		// Always make progress. Keeping the offending character in an Unknown
		// token lets the parser report and recover from it instead of hanging.
		token = l.createTokenCount(Unknown, leadingWhitespace, 1)
	}

	return token
}

func (l *Lexer) createStringToken(leadingWhitespace string) Token {
	startPos := l.pos
	startCol := l.currentColumn
	_ = l.advance()
	terminated := false
	for l.pos < len(l.input) && l.getCurrentChar() != '\n' {
		if l.getCurrentChar() == '"' {
			// CMC escapes a double quote inside a string as '"'.
			if l.pos > startPos && l.input[l.pos-1] == '\'' && l.getNextChar() == '\'' {
				_ = l.advance()
				continue
			}
			_ = l.advance()
			terminated = true
			break
		}
		_ = l.advance()
	}
	lexeme := string(l.input[startPos:l.pos])
	if !terminated {
		l.diagnostics = append(l.diagnostics, diag.Diagnostic{
			Kind: diag.StringUnterminated, Severity: diag.Error,
			Range: source.NewRange(l.currentLine, startCol, len([]rune(lexeme))),
		})
	}
	l.lastTokenKind = LiteralString
	return Token{Kind: LiteralString, Lexeme: lexeme, LeadingWhitespace: leadingWhitespace, Range: source.NewRange(l.currentLine, startCol, len([]rune(lexeme)))}
}

func (l *Lexer) createHexToken(leadingWhitespace string) Token {
	startPos := l.pos
	startCol := l.currentColumn
	_ = l.advanceCount(2)
	for l.pos < len(l.input) {
		current := l.getCurrentChar()
		if (current >= '0' && current <= '9') || (unicode.ToLower(current) >= 'a' && unicode.ToLower(current) <= 'f') {
			_ = l.advance()
			continue
		}
		if current == '$' && l.getNextChar() == '(' {
			depth := 0
			for l.pos < len(l.input) && l.getCurrentChar() != '\n' {
				value := l.getCurrentChar()
				_ = l.advance()
				if value == '(' {
					depth++
				} else if value == ')' {
					depth--
					if depth == 0 {
						break
					}
				}
			}
			continue
		}
		break
	}
	lexeme := string(l.input[startPos:l.pos])
	l.lastTokenKind = LiteralHex
	return Token{Kind: LiteralHex, Lexeme: lexeme, LeadingWhitespace: leadingWhitespace, Range: source.NewRange(l.currentLine, startCol, len([]rune(lexeme)))}
}

func (l *Lexer) createPreprocessorToken(leadingWhitespace string) Token {
	token := l.createTokeWhile(PreprocessorUnknown, leadingWhitespace, func(r rune) bool {
		return r != ' ' && r != '\n'
	})

	if strings.EqualFold(token.Lexeme, "#include") {
		token.Kind = PreprocessorInclude
	}

	return token
}

func (l *Lexer) createIdentifierToken(leadingWhitespace string) Token {
	currentChar := l.getCurrentChar()
	nextChar := l.getNextChar()
	atLineStart := l.lastTokenKind == NewLine
	if l.lastTokenKind == LiteralNumber && unicode.ToLower(currentChar) == 'e' {
		if unicode.ToLower(nextChar) == 'x' {
			return l.createTokenCount(LiteralNumberEx, leadingWhitespace, 2)
		}
		// SINAMICS .tea exports use standard E notation while CMC scripts also
		// accept the documented EX spelling.
		return l.createTokenCount(LiteralNumberEx, leadingWhitespace, 1)
	}

	token := l.createTokeWhile(LiteralIdentifier, leadingWhitespace, func(r rune) bool {
		return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r)
	})

	switch {
	case strings.EqualFold(token.Lexeme, "if"):
		token.Kind = KeywordIf
	case strings.EqualFold(token.Lexeme, "else"):
		token.Kind = KeywordElse
	case strings.EqualFold(token.Lexeme, "elsif"):
		token.Kind = KeywordElseIf
	case strings.EqualFold(token.Lexeme, "elif"):
		token.Kind = KeywordElseIf
	case strings.EqualFold(token.Lexeme, "endif"):
		token.Kind = KeywordEndIf
	case strings.EqualFold(token.Lexeme, "while"):
		token.Kind = KeywordWhile
	case strings.EqualFold(token.Lexeme, "endwhile"):
		token.Kind = KeywordEndWhile
	case strings.EqualFold(token.Lexeme, "null"):
		token.Kind = LiteralNull
	case strings.EqualFold(token.Lexeme, "true"):
		token.Kind = LiteralTrue
	case strings.EqualFold(token.Lexeme, "false"):
		token.Kind = LiteralFalse
	case strings.EqualFold(token.Lexeme, "proc"):
		token.Kind = KeywordProc
	case strings.EqualFold(token.Lexeme, "func"):
		token.Kind = KeywordFunc
	case strings.EqualFold(token.Lexeme, "return"):
		token.Kind = KeywordReturn
	case strings.EqualFold(token.Lexeme, "nc"):
		token.Kind = KeywordNamespaceNc
	case strings.EqualFold(token.Lexeme, "ps"):
		token.Kind = KeywordNamespacePs
	case strings.EqualFold(token.Lexeme, "bd"):
		token.Kind = KeywordNamespaceBd
	case strings.EqualFold(token.Lexeme, "chandata"):
		token.Kind = KeywordNamespaceChan
	}

	if token.Kind == LiteralIdentifier && atLineStart && isBlockNumber(token.Lexeme) {
		token.Kind = LiteralBlockNumber
	}

	l.lastTokenKind = token.Kind

	return token
}

func isBlockNumber(value string) bool {
	if len(value) < 2 || (value[0] != 'N' && value[0] != 'n') {
		return false
	}
	for _, r := range value[1:] {
		if !unicode.IsNumber(r) {
			return false
		}
	}
	return true
}

func (l *Lexer) createDelimitedToken(
	kind TokenKind,
	diagKind diag.DiagnosticKind,
	leadingWhitespace string,
	endChar rune,
) Token {
	if l.input[l.pos] == '\n' {
		panic("must not be called with newline as the current char")
	}
	if endChar == '\n' {
		panic("must not be called with newline as the end char")
	}

	endFunct := func(c rune) bool {
		return c != endChar
	}

	startPos := l.pos
	startCol := l.currentColumn
	// this cannot error now, since the current char is not a newline
	_ = l.advance()
	err := l.advancePosWhile(endFunct)
	if err != nil {
		l.diagnostics = append(l.diagnostics, diag.Diagnostic{
			Kind:     diagKind,
			Severity: diag.Error,
			Range:    source.NewRange(l.currentLine, startCol, l.currentColumn-startCol),
		})
	} else {
		// this can also not error not error, since endchar is not a newline
		_ = l.advance()
	}

	lexeme := string(l.input[startPos:l.pos])
	return Token{
		Kind:              kind,
		Lexeme:            string(l.input[startPos:l.pos]),
		LeadingWhitespace: leadingWhitespace,
		Range:             source.NewRange(l.currentLine, startCol, len([]rune(lexeme))),
	}
}

func (l *Lexer) getNextChar() rune {
	if l.pos+1 < len(l.input) {
		return l.input[l.pos+1]
	}

	return eof
}

func (l *Lexer) getCurrentChar() rune {
	if l.pos < len(l.input) {
		return l.input[l.pos]
	}

	return eof
}

func (l *Lexer) createTokenCount(kind TokenKind, leadingWhitespace string, count int) Token {
	l.lastTokenKind = kind
	startPos := l.pos
	startCol := l.currentColumn
	err := l.advanceCount(count)
	if err != nil {
		fmt.Printf("createTokenCount advance error: %v\n", err)
	}

	lexeme := string(l.input[startPos:l.pos])
	return Token{
		Kind:              kind,
		Lexeme:            lexeme,
		LeadingWhitespace: leadingWhitespace,
		Range:             source.NewRange(l.currentLine, startCol, len([]rune(lexeme))),
	}
}

func (l *Lexer) createTokeWhile(
	kind TokenKind,
	leadingWhitespace string,
	predicate func(rune) bool,
) Token {
	if l.input[l.pos] == '\n' {
		panic("must not be called with newline as the current char")
	}

	l.lastTokenKind = kind
	startPos := l.pos
	startCol := l.currentColumn
	// this cannot error now, since the current char is not a newline
	_ = l.advance()
	// this also cannot error, because of the predicate
	_ = l.advancePosWhile(predicate)
	end := l.pos

	lexeme := string(l.input[startPos:end])
	return Token{
		Kind:              kind,
		Lexeme:            lexeme,
		LeadingWhitespace: leadingWhitespace,
		Range:             source.NewRange(l.currentLine, startCol, len([]rune(lexeme))),
	}
}

func (l *Lexer) advancePosWhile(
	predicate func(rune) bool,
) error {
	for _, c := range l.input[l.pos:] {
		if !predicate(c) {
			return nil
		}
		if c == '\n' {
			return errors.New("unexpected newline")
		}
		l.pos++
		l.currentColumn++
	}
	return errors.New("eof")
}

func (l *Lexer) advanceCount(count int) error {
	if count == 0 {
		return nil
	}

	if l.pos+count > len(l.input) {
		return errors.New("EOF")
	}

	if count > 1 && slices.Contains(l.input[l.pos:l.pos+count-1], '\n') {
		return errors.New("unexpected newline")
	}

	l.pos += count
	l.currentColumn += count
	return nil
}

func (l *Lexer) advance() error {
	return l.advanceCount(1)
}
