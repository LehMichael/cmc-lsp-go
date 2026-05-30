package diag

import (
	"github.com/lehmichael/cmc-lsp-go/internal/source"
)

type Diagnostic struct {
	Kind     DiagnosticKind
	Range    source.SourceRange
	Severity DiagnosticSeverity
}

//go:generate go tool enumer -type=DiagnosticKind -json
type DiagnosticKind int

const (
	UnexpectedToken DiagnosticKind = iota
	FunctionInvalidIdentifier
	FunctionInvalidParameterDef
	FunctionInvalidOpeningBrace
	FunctionBodyClosingBraceMissing
	SectionInvalidDrive
	SectionInvalidChan
	SectionFormatUnrecogniced
	PreprocessorUnknown
	ExpressionGroupedMissingClosingParentheses
	NumberLiteralInvalidExponent
	NumberLiteralParseError
	BicoLiteralParseError
	VersionLiteralParserError
	ExpressionInvalid
	BitwiseLiteralInvalidHex
	BitwiseLiteralInvalidBin
	ExpressionReplacementMissingClosingParentheses
	WhileEndMissing
	IfThenEndMissing
	IfElseIfEndMissing
	IfElseEndMissing
	CallStatementMalformed
	InvaliStatement

	// lexer
	SectionUnterminated
	StringUnterminated
	NumberFormatUnterminated
)

//go:generate go tool enumer -type=DiagnosticSeverity -json
type DiagnosticSeverity int

const (
	Error DiagnosticSeverity = iota
)
