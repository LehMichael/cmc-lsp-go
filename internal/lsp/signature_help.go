package lsp

import (
	"strings"

	"github.com/lehmichael/cmc-lsp-go/internal/lexer"
)

type callContext struct {
	name            string
	activeParameter int
	delimiter       lexer.TokenKind
}

func (server *Lsp) handleSignatureHelp(message requestMessage) {
	var params textDocumentPositionParams
	if err := decodeParams(message.Params, &params); err != nil {
		server.respond(message.ID, nil, &responseError{Code: InvalidParams, Message: err.Error()})
		return
	}
	text, err := server.overlay.ReadURI(params.TextDocument.URI)
	if err != nil {
		server.respond(message.ID, nil, nil)
		return
	}
	context, ok := callAt(text, params.Position)
	if !ok {
		server.respond(message.ID, nil, nil)
		return
	}
	definition := server.callableDefinitionAt(params.TextDocument.URI, context.name)
	if definition == nil {
		server.respond(message.ID, nil, nil)
		return
	}

	labels := callableArgumentLabels(definition.ArgumentCount, definition.Arguments)
	parameters := make([]parameterInformation, len(labels))
	for index, label := range labels {
		parameters[index].Label = label
		if index < len(definition.Arguments) && definition.Arguments[index] != "" {
			parameters[index].Documentation = &markupContent{Kind: "markdown", Value: definition.Arguments[index]}
		}
	}
	activeParameter := context.activeParameter
	if len(parameters) == 0 {
		activeParameter = 0
	} else if activeParameter >= len(parameters) {
		activeParameter = len(parameters) - 1
	}
	var documentation *markupContent
	if definition.Description != "" {
		documentation = &markupContent{Kind: "markdown", Value: definition.Description}
	}
	server.respond(message.ID, signatureHelp{
		Signatures: []signatureInformation{{
			Label:         definition.Name + "(" + strings.Join(labels, ", ") + ")",
			Documentation: documentation,
			Parameters:    parameters,
		}},
		ActiveParameter: activeParameter,
	}, nil)
}

func callableArgumentLabels(count int, descriptions []string) []string {
	labels := make([]string, count)
	for index := range labels {
		labels[index] = "$"
		if index < len(descriptions) && descriptions[index] != "" {
			labels[index] = descriptions[index]
		}
	}
	return labels
}

// callAt finds the innermost open call at the cursor. Token-based tracking also
// works while the user is midway through an incomplete call, before the parser
// can construct a complete expression.
func callAt(text string, target position) (callContext, bool) {
	tokens, _ := lexer.Tokenize(text)
	stack := make([]callContext, 0)
	var previous *lexer.Token
	var beforePrevious *lexer.Token
	for index := range tokens {
		token := &tokens[index]
		tokenRange := sourceRangeToLSP(text, token.Range)
		if positionInRange(target, tokenRange) && (token.Kind == lexer.Comment || token.Kind == lexer.LiteralString) {
			return callContext{}, false
		}
		if !positionLess(tokenRange.Start, target) {
			break
		}
		switch token.Kind {
		case lexer.SymbolLeftParen:
			context := callContext{delimiter: lexer.SymbolRightParen}
			isDefinition := beforePrevious != nil && (beforePrevious.Kind == lexer.KeywordFunc || beforePrevious.Kind == lexer.KeywordProc)
			if previous != nil && previous.Kind == lexer.LiteralIdentifier && !isDefinition {
				context.name = previous.Lexeme
			}
			stack = append(stack, context)
		case lexer.SymbolLeftBracket:
			stack = append(stack, callContext{delimiter: lexer.SymbolRightBracket})
		case lexer.SymbolComma:
			if len(stack) > 0 {
				stack[len(stack)-1].activeParameter++
			}
		case lexer.SymbolRightParen, lexer.SymbolRightBracket:
			if len(stack) > 0 && stack[len(stack)-1].delimiter == token.Kind {
				stack = stack[:len(stack)-1]
			}
		}

		switch token.Kind {
		case lexer.NewLine, lexer.Comment:
			previous = nil
			beforePrevious = nil
		default:
			beforePrevious = previous
			copy := *token
			previous = &copy
		}
	}
	for index := len(stack) - 1; index >= 0; index-- {
		if stack[index].name != "" {
			return stack[index], true
		}
	}
	return callContext{}, false
}

func positionLess(left, right position) bool {
	return left.Line < right.Line || left.Line == right.Line && left.Character < right.Character
}

func positionInRange(target position, value lspRange) bool {
	return !positionLess(target, value.Start) && positionLess(target, value.End)
}
