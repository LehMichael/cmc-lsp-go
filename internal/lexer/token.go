package lexer

import "github.com/lehmichael/cmc-lsp-go/internal/source"

//go:generate go tool enumer -type=TokenKind -json
type TokenKind int

const (
	Unknown TokenKind = iota

	KeywordIf
	KeywordElse
	KeywordElseIf
	KeywordEndIf
	KeywordWhile
	KeywordEndWhile
	KeywordProc
	KeywordFunc
	KeywordReturn

	KeywordNamespaceNc
	KeywordNamespacePs
	KeywordNamespaceBd
	KeywordNamespaceChan

	Section

	OperatorAssign
	OperatorAssignRaw
	OperatorAssignIfBlank
	OperatorAddAssign
	OperatorSubtractAssign
	OperatorMultiplyAssign
	OperatorDivideAssign
	OperatorOrAssign
	OperatorAndAssign
	OperatorDelete

	OperatorEqual
	OperatorUnequal
	OperatorLessThan
	OperatorLessThanEqual
	OperatorGreaterThan
	OperatorGreaterThanEqual

	OperatorLogAnd
	OperatorLogOr

	OperatorStringConcat

	OperatorAdd
	OperatorSubtract
	OperatorMultiply
	OperatorDivide
	OperatorAnd
	OperatorOr
	OperatorNegate

	LiteralIdentifier
	LiteralBlockNumber
	LiteralString

	LiteralNumberFormat

	LiteralNumber
	LiteralHex
	LiteralNumberEx
	LiteralNull
	LiteralTrue
	LiteralFalse

	SymbolLeftParen
	SymbolRightParen
	SymbolLeftBracket
	SymbolRightBracket
	SymbolLeftBrace
	SymbolRightBrace
	SymbolDot
	SymbolComma
	SymbolDollar
	SymbolDollarParen

	Comment

	PreprocessorInclude
	PreprocessorUnknown

	EOF
	NewLine
)

type Token struct {
	Kind              TokenKind
	Lexeme            string
	LeadingWhitespace string
	Range             source.SourceRange
}
