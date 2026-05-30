package diag

import (
	"github.com/lehmichael/cmc-lsp-go/internal/source"
)

type Diagnostic struct {
	Kind     DiagnosticKind
	Range    source.SourceRange
	Severity DiagnosticSeverity
}

type DiagnosticKind int

const (
	DiagUnexpectedToken DiagnosticKind = iota
	DiagFunctionInvalidIdentifier
	DiagFunctionInvalidParameterDef
	DiagFunctionInvalidOpeningBrace
	DiagFunctionBodyClosingBraceMissing
	DiagSectionInvalidDrive
	DiagSectionInvalidChan
	DiagSectionFormatUnrecogniced
	DiagPreprocessorUnknown
	DiagExpressionGroupedMissingClosingParentheses
	DiagNumberLiteralInvalidExponent
	DiagNumberLiteralParseError
	DiagBicoLiteralParseError
	DiagVersionLiteralParserError
	DiagExpressionInvalid
	DiagBitwiseLiteralInvalidHex
	DiagBitwiseLiteralInvalidBin
	DiagExpressionReplacementMissingClosingParentheses
	DiagWhileEndMissing
	DiagIfThenEndMissing
	DiagIfElseIfEndMissing
	DiagIfElseEndMissing
	DiagCallStatementMalformed
	DiagInvaliStatement

	// lexer
	SectionUnterminated
	StringUnterminated
	NumberFormatUnterminated
)

type DiagnosticSeverity int

const (
	SeverityError DiagnosticSeverity = iota
)
