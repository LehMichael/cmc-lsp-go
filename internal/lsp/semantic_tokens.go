package lsp

import (
	"strings"
	"unicode/utf16"

	"github.com/lehmichael/cmc-lsp-go/internal/lexer"
)

// The order of these entries is part of the wire format: encoded tokens refer
// to their indexes in this legend.
var semanticTokenTypes = []string{
	"namespace",
	"parameter",
	"variable",
	"property",
	"function",
	"macro",
	"keyword",
	"comment",
	"string",
	"number",
	"operator",
}

var semanticTokenModifiers = []string{"declaration"}

const (
	semanticNamespace uint32 = iota
	semanticParameter
	semanticVariable
	semanticProperty
	semanticFunction
	semanticMacro
	semanticKeyword
	semanticComment
	semanticString
	semanticNumber
	semanticOperator
)

const semanticDeclaration uint32 = 1 << 0

func (server *Lsp) handleSemanticTokens(message requestMessage) {
	var params documentParams
	if err := decodeParams(message.Params, &params); err != nil {
		server.respond(message.ID, nil, &responseError{Code: InvalidParams, Message: err.Error()})
		return
	}
	text, err := server.overlay.ReadURI(params.TextDocument.URI)
	if err != nil {
		server.respond(message.ID, nil, &responseError{Code: InvalidParams, Message: err.Error()})
		return
	}
	server.respond(message.ID, semanticTokens{Data: semanticTokensFor(text)}, nil)
}

func semanticTokensFor(text string) []uint32 {
	tokens, _ := lexer.Tokenize(text)
	var result []uint32
	previousLine := 0
	previousStart := 0
	havePrevious := false

	for index, token := range tokens {
		tokenType, modifiers, ok := classifySemanticToken(tokens, index)
		if !ok {
			continue
		}
		start := sourceRangeToLSP(text, token.Range).Start
		length := utf16Length(token.Lexeme)
		if length == 0 {
			continue
		}
		deltaLine := start.Line
		deltaStart := start.Character
		if havePrevious {
			deltaLine -= previousLine
			if deltaLine == 0 {
				deltaStart -= previousStart
			}
		}
		result = append(result, uint32(deltaLine), uint32(deltaStart), uint32(length), tokenType, modifiers)
		previousLine = start.Line
		previousStart = start.Character
		havePrevious = true
	}
	return result
}

func classifySemanticToken(tokens []lexer.Token, index int) (uint32, uint32, bool) {
	token := tokens[index]
	switch token.Kind {
	case lexer.KeywordNamespaceNc, lexer.KeywordNamespacePs,
		lexer.KeywordNamespaceBd, lexer.KeywordNamespaceChan:
		return semanticNamespace, 0, true
	case lexer.KeywordIf, lexer.KeywordElse, lexer.KeywordElseIf,
		lexer.KeywordEndIf, lexer.KeywordWhile, lexer.KeywordEndWhile,
		lexer.KeywordProc, lexer.KeywordFunc, lexer.KeywordReturn,
		lexer.LiteralNull, lexer.LiteralTrue, lexer.LiteralFalse:
		return semanticKeyword, 0, true
	case lexer.Section:
		return semanticProperty, 0, true
	case lexer.Comment:
		return semanticComment, 0, true
	case lexer.LiteralString, lexer.LiteralNumberFormat:
		return semanticString, 0, true
	case lexer.LiteralNumber, lexer.LiteralHex, lexer.LiteralNumberEx, lexer.LiteralBlockNumber:
		return semanticNumber, 0, true
	case lexer.PreprocessorInclude, lexer.PreprocessorUnknown:
		return semanticMacro, 0, true
	case lexer.LiteralIdentifier:
		return classifyIdentifier(tokens, index)
	default:
		if isSemanticOperator(token.Kind) {
			return semanticOperator, 0, true
		}
		return 0, 0, false
	}
}

func classifyIdentifier(tokens []lexer.Token, index int) (uint32, uint32, bool) {
	previous := lexer.Unknown
	next := lexer.Unknown
	if index > 0 {
		previous = tokens[index-1].Kind
	}
	if index+1 < len(tokens) {
		next = tokens[index+1].Kind
	}

	if previous == lexer.KeywordFunc || previous == lexer.KeywordProc {
		return semanticFunction, semanticDeclaration, true
	}
	if next == lexer.SymbolLeftParen {
		return semanticFunction, 0, true
	}
	if previous == lexer.SymbolDot {
		return semanticProperty, 0, true
	}
	if next == lexer.SymbolDot || isCMCNamespace(tokens[index].Lexeme) {
		return semanticNamespace, 0, true
	}
	if previous == lexer.SymbolDollar {
		return semanticParameter, 0, true
	}
	return semanticVariable, 0, true
}

func isCMCNamespace(value string) bool {
	switch strings.ToLower(value) {
	case "up", "tmp", "gud", "ipo", "nck":
		return true
	default:
		return false
	}
}

func utf16Length(value string) int {
	return len(utf16.Encode([]rune(value)))
}

func isSemanticOperator(kind lexer.TokenKind) bool {
	switch kind {
	case lexer.OperatorAssign, lexer.OperatorAssignRaw, lexer.OperatorAssignIfBlank,
		lexer.OperatorAddAssign, lexer.OperatorSubtractAssign, lexer.OperatorMultiplyAssign,
		lexer.OperatorDivideAssign, lexer.OperatorOrAssign, lexer.OperatorAndAssign,
		lexer.OperatorDelete, lexer.OperatorEqual, lexer.OperatorUnequal,
		lexer.OperatorLessThan, lexer.OperatorLessThanEqual, lexer.OperatorGreaterThan,
		lexer.OperatorGreaterThanEqual, lexer.OperatorLogAnd, lexer.OperatorLogOr,
		lexer.OperatorStringConcat, lexer.OperatorAdd, lexer.OperatorSubtract,
		lexer.OperatorMultiply, lexer.OperatorDivide, lexer.OperatorAnd,
		lexer.OperatorOr, lexer.OperatorNegate:
		return true
	default:
		return false
	}
}
