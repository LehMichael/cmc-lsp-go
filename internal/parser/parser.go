package parser

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/lehmichael/cmc-lsp-go/internal/diag"
	l "github.com/lehmichael/cmc-lsp-go/internal/lexer"
	"github.com/lehmichael/cmc-lsp-go/internal/source"
)

func Parse(tokens []l.Token, diagnostics []diag.Diagnostic) (Ast, []diag.Diagnostic) {
	p := parser{
		Tokens:      tokens,
		Pos:         0,
		Diagnostics: diagnostics,
	}

	var statements Ast

	for p.currentToken().Kind != l.EOF {
		// fmt.Printf("a: %v\n", p.currentToken().Kind.String())
		comments := p.parseCommentBlock()
		if p.currentToken().Kind == l.EOF {
			break
		}
		statement := p.parseStatement(comments)
		statements = append(statements, statement)
	}

	return statements, p.Diagnostics
}

type parser struct {
	Tokens      []l.Token
	Pos         int
	Diagnostics []diag.Diagnostic
}

func (p *parser) currentToken() l.Token {
	return p.Tokens[p.Pos]
}

func (p *parser) peekToken(offset int) l.Token {
	position := p.Pos + offset
	if position >= len(p.Tokens) {
		return p.Tokens[len(p.Tokens)-1]
	}
	return p.Tokens[position]
}

func (p *parser) advance() l.Token {
	if p.Pos < len(p.Tokens)-1 {
		p.Pos++
	}
	return p.currentToken()
}

func (p *parser) expect(kinds []l.TokenKind) *l.Token {
	token := p.currentToken()
	if slices.Contains(kinds, token.Kind) {
		return &token
	}

	return nil
}

func (p *parser) expectAndAdvance(kinds []l.TokenKind) *l.Token {
	if p.expect(kinds) != nil {
		token := p.currentToken()
		_ = p.advance()
		return &token
	}

	return nil
}

func (p *parser) parseStatement(leadingComments []string) Statement {
	token := p.currentToken()
	// startPos := p.Pos

	switch token.Kind {
	case l.KeywordIf:
		return p.parseIfBlock(leadingComments)
	case l.KeywordWhile:
		return p.parseWhileBlock(leadingComments)
	case l.Section, l.KeywordNamespaceChan:
		return p.parseSectionStatement(leadingComments)
	case l.KeywordNamespaceNc, l.KeywordNamespacePs, l.KeywordNamespaceBd:
		return p.parseNamespacedStatement(leadingComments)
	case l.LiteralBlockNumber:
		_ = p.advance()
		if p.currentToken().Kind == l.LiteralIdentifier || p.currentToken().Kind == l.SymbolDollarParen {
			return p.parseIdentifierStatement(leadingComments)
		}
		return p.parseInvalidStatement(leadingComments)
	case l.LiteralIdentifier:
		return p.parseIdentifierStatement(leadingComments)
	case l.KeywordReturn:
		return p.parseReturnCall(leadingComments)
	case l.PreprocessorInclude:
		return p.parsePreprocessor(leadingComments)
	case l.KeywordProc, l.KeywordFunc:
		return p.parseFunctionStatement(leadingComments)
	case l.NewLine:
		_ = p.advance()
		return Statement{
			Kind:            NewLine{},
			LeadingComments: leadingComments,
			Range:           token.Range,
		}
	}

	return p.parseInvalidStatement(leadingComments)
}

func (p *parser) parseReturnCall(leadingComments []string) Statement {
	token := p.currentToken()
	identifier := IdentifierExpression{
		Segments: []IdentifierSegment{{Parts: []IdentifierPart{LiteralIdentifier(token.Lexeme)}}},
		Range:    token.Range,
	}
	_ = p.advance()
	if p.currentToken().Kind != l.SymbolLeftParen {
		return p.parseInvalidStatement(leadingComments)
	}
	return p.parseCallStatement(identifier, leadingComments)
}

func (p *parser) parseFunctionStatement(leadingComments []string) Statement {
	startToken := p.currentToken()
	if startToken.Kind != l.KeywordFunc && startToken.Kind != l.KeywordProc {
		panic("must be called with function or proc token")
	}

	var kind FunctionKind
	switch startToken.Kind {
	case l.KeywordProc:
		kind = Procedure
	case l.KeywordFunc:
		kind = Function
	default:
		panic("must be called with function or proc token")

	}

	_ = p.advance()

	var invalidIdentTokens []l.Token
	var identifier *IdentifierExpression
	if p.currentToken().Kind == l.LiteralIdentifier ||
		p.currentToken().Kind == l.SymbolDollarParen {
		i := p.parseIdentifier(nil)
		identifier = &i
	} else {
		invalidIdentTokens = p.recoverFromError(
			diag.FunctionInvalidIdentifier,
			p.Pos,
			[]l.TokenKind{l.SymbolLeftParen, l.NewLine, l.SymbolLeftBrace, l.Comment, l.EOF},
		)
	}

	paramTokenPos := p.Pos
	invalidParams := false

	if t := p.expectAndAdvance([]l.TokenKind{l.SymbolLeftParen}); t == nil {
		invalidParams = true
	}

	arcCount := 0

	if !invalidParams {
		if p.currentToken().Kind == l.SymbolRightParen {
			_ = p.advance()
		} else {
		loop:
			for {
				if t := p.expectAndAdvance([]l.TokenKind{l.SymbolDollar}); t == nil {
					break
				}
				arcCount++
				switch p.currentToken().Kind {
				case l.SymbolComma:
					_ = p.advance()
				case l.SymbolRightParen:
					_ = p.advance()
					break loop
				default:
					arcCount = 0
					invalidParams = true
					break loop
				}
			}
		}
	}

	var invalidParamTokens []l.Token
	if invalidParams {
		invalidParamTokens = p.recoverFromError(
			diag.FunctionInvalidParameterDef,
			paramTokenPos,
			[]l.TokenKind{l.SymbolLeftBrace, l.SymbolRightParen, l.NewLine, l.Comment, l.EOF},
		)

		if p.currentToken().Kind == l.SymbolRightParen {
			invalidParamTokens = append(invalidParamTokens, p.currentToken())
			_ = p.advance()
		}
	}

	noOpeningBrace := false
	var invalidOpeningBraceTokens []l.Token
	if t := p.expectAndAdvance([]l.TokenKind{l.SymbolLeftBrace}); t == nil {
		noOpeningBrace = true
		invalidOpeningBraceTokens = p.recoverFromError(
			diag.FunctionInvalidBeforeOpeningBrace,
			p.Pos,
			[]l.TokenKind{l.SymbolLeftBrace, l.Comment, l.NewLine, l.EOF},
		)
		if p.currentToken().Kind == l.SymbolLeftBrace {
			noOpeningBrace = false
			_ = p.advance()
		}
	}

	if noOpeningBrace {
		p.Diagnostics = append(p.Diagnostics, diag.Diagnostic{
			Kind:  diag.FunctionMissingOpeningBrace,
			Range: p.currentToken().Range,
		})
	}

	invalidAfterOpeningBraceTokens, trailingCommentStart := p.parseEndComment()
	// consume newline
	_ = p.advance()

	var comments []string
	var body []Statement
	endToken := p.currentToken()
	var trailingCommentEnd *string
	var invalidAfterClosingBraceTokens []l.Token

	if !noOpeningBrace {
		body, comments = p.parseBody(
			[]l.TokenKind{l.SymbolRightBrace},
			diag.FunctionBodyClosingBraceMissing,
		)

		endToken = p.currentToken()
		_ = p.advance()

		invalidAfterClosingBraceTokens, trailingCommentEnd = p.parseEndComment()
	}

	leadingCommentsEnd := comments
	description, argumentDescriptions := parseFunctionDocumentation(leadingComments)

	return Statement{
		Kind: FunctionStatement{
			Kind:                      kind,
			Identifier:                identifier,
			Description:               description,
			ArgumentDescriptions:      argumentDescriptions,
			ArgCount:                  arcCount,
			Body:                      body,
			TrailingCommentStart:      trailingCommentStart,
			LeadingCommentsEnd:        leadingCommentsEnd,
			InvalidIdentTokens:        invalidIdentTokens,
			InvalidParamTokens:        invalidParamTokens,
			InvalidAfterOpeningBrace:  invalidAfterOpeningBraceTokens,
			InvalidBeforeOpeningBrace: invalidOpeningBraceTokens,
			MissingOpeningBrace:       noOpeningBrace,
		},
		LeadingComments: leadingComments,
		InvalidAfter:    invalidAfterClosingBraceTokens,
		TrailingComment: trailingCommentEnd,
		Range:           source.MergeRange(startToken.Range, endToken.Range),
	}
}

func parseFunctionDocumentation(comments []string) (string, []string) {
	var description string
	arguments := make([]string, 9)
	highestArgument := 0

	for _, comment := range comments {
		content := strings.TrimSpace(comment)
		content = strings.TrimSpace(strings.TrimPrefix(content, ";"))
		key, value, found := strings.Cut(content, ":")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if strings.EqualFold(key, "Description") {
			description = value
			continue
		}
		if len(key) < 4 || !strings.EqualFold(key[:3], "Arg") {
			continue
		}
		position, err := strconv.Atoi(key[3:])
		if err != nil || position < 1 || position > len(arguments) {
			continue
		}
		arguments[position-1] = value
		if position > highestArgument {
			highestArgument = position
		}
	}

	return description, arguments[:highestArgument]
}

func (p *parser) parsePreprocessor(leadingComments []string) Statement {
	startToken := p.currentToken()
	if startToken.Kind != l.PreprocessorInclude {
		panic("must be called with preprocessor token")
	}

	var endToken l.Token
	var kind PreprocessorStatementKind

	switch startToken.Kind {
	case l.PreprocessorInclude:
		_ = p.advance()
		var invalidToken *l.Token
		var path string
		if t := p.expect([]l.TokenKind{l.LiteralString}); t != nil {
			endToken = *t
			path = t.Lexeme
			_ = p.advance()
		} else {
			token := p.currentToken()
			invalidToken = &token
			endToken = token
			p.Diagnostics = append(p.Diagnostics, diag.Diagnostic{
				Kind: diag.UnexpectedToken, Range: token.Range, Severity: diag.Error,
			})
			if token.Kind != l.EOF && token.Kind != l.NewLine && token.Kind != l.Comment {
				_ = p.advance()
			}
		}

		kind = IncludePpStatement{
			Path:         path,
			InvalidToken: invalidToken,
		}
	}

	invalidAfter, trailingComment := p.parseEndComment()

	return Statement{
		Kind: PreprocessorStatement{
			Kind: kind,
		},
		LeadingComments: leadingComments,
		TrailingComment: trailingComment,
		InvalidAfter:    invalidAfter,
		Range:           source.MergeRange(startToken.Range, endToken.Range),
	}
}

func (p *parser) parseIdentifierStatement(leadingComments []string) Statement {
	identifier := p.parseIdentifier(nil)

	token := p.currentToken()

	isAssignment := tokenToAssignmentKind(token) != nil

	switch {
	case token.Kind == l.SymbolLeftParen:
		return p.parseCallStatement(identifier, leadingComments)
	case token.Kind == l.OperatorDelete:
		return p.parseDeleteStatement(identifier, leadingComments)
	case isAssignment:
		return p.parseAssignment(identifier, leadingComments)
	}

	return p.parseInvalidStatement(leadingComments)
}

func (p *parser) parseEndComment() ([]l.Token, *string) {
	var invalidAfter []l.Token
	if t := p.expect([]l.TokenKind{l.NewLine, l.EOF, l.Comment}); t == nil {
		invalidAfter = p.recoverFromError(
			diag.UnexpectedToken,
			p.Pos,
			[]l.TokenKind{l.NewLine, l.EOF, l.Comment},
		)
	}
	trailingComment := p.parseOptionalComment()

	return invalidAfter, trailingComment
}

func (p *parser) parseInvalidStatement(leadingComments []string) Statement {
	tokens, trailingComment := p.parseEndComment()
	if len(tokens) == 0 {
		token := p.currentToken()
		p.Diagnostics = append(p.Diagnostics, diag.Diagnostic{Kind: diag.InvaliStatement, Range: token.Range, Severity: diag.Error})
		return Statement{Kind: InvalidStatement{}, Range: token.Range, LeadingComments: leadingComments, TrailingComment: trailingComment}
	}

	return Statement{
		Kind:            InvalidStatement(tokens),
		Range:           source.MergeRange(tokens[0].Range, tokens[len(tokens)-1].Range),
		LeadingComments: leadingComments,
		TrailingComment: trailingComment,
	}
}

func (p *parser) parseAssignment(
	identifier IdentifierExpression,
	leadingComments []string,
) Statement {
	op := tokenToAssignmentKind(p.currentToken())
	if op == nil {
		panic("must be called with assignment token")
	}
	_ = p.advance()

	var value Expression
	if *op == AssignRaw {
		start := p.currentToken()
		var raw strings.Builder
		end := start
		for p.currentToken().Kind != l.NewLine && p.currentToken().Kind != l.Comment && p.currentToken().Kind != l.EOF {
			t := p.currentToken()
			raw.WriteString(t.LeadingWhitespace)
			raw.WriteString(t.Lexeme)
			end = t
			_ = p.advance()
		}
		value = Expression{Kind: RawLiteral(strings.TrimSpace(raw.String())), Range: source.MergeRange(start.Range, end.Range)}
	} else {
		value = p.parseExpression(0)
	}

	invalidAfter, trailingComment := p.parseEndComment()

	return Statement{
		Kind: Assignment{
			Kind:   *op,
			Target: identifier,
			Value:  value,
		},
		Range:           source.MergeRange(identifier.Range, value.Range),
		LeadingComments: leadingComments,
		TrailingComment: trailingComment,
		InvalidAfter:    invalidAfter,
	}
}

func (p *parser) parseDeleteStatement(
	identifier IdentifierExpression,
	leadingComments []string,
) Statement {
	endToken := p.currentToken()

	if endToken.Kind != l.OperatorDelete {
		panic("must be called with SymbolLeftParen token")
	}
	_ = p.advance()

	var invalidAfter []l.Token
	if t := p.expect([]l.TokenKind{l.NewLine, l.EOF, l.Comment}); t == nil {
		invalidAfter = p.recoverFromError(
			diag.UnexpectedToken,
			p.Pos,
			[]l.TokenKind{l.NewLine, l.EOF, l.Comment},
		)
	}
	trailingComment := p.parseOptionalComment()
	if t := p.expectAndAdvance([]l.TokenKind{l.NewLine, l.EOF}); t == nil {
		invalidAfter = p.recoverFromError(
			diag.UnexpectedToken,
			p.Pos,
			[]l.TokenKind{l.NewLine, l.EOF},
		)
		_ = p.advance()
	}

	return Statement{
		Kind: DeleteStatement{
			Identifier: identifier,
		},
		Range:           source.MergeRange(identifier.Range, endToken.Range),
		LeadingComments: leadingComments,
		TrailingComment: trailingComment,
		InvalidAfter:    invalidAfter,
	}
}

func (p *parser) parseCallExpression(identifier IdentifierExpression) Expression {
	startToken := p.currentToken()
	if startToken.Kind != l.SymbolLeftParen {
		panic("must be called with SymbolLeftParen token")
	}
	_ = p.advance()

	var parameters []Expression

	if p.currentToken().Kind != l.SymbolRightParen {
		for {
			parameters = append(parameters, p.parseExpression(0))

			if p.currentToken().Kind == l.SymbolComma {
				_ = p.advance()
				continue
			}
			break
		}
	}

	missingClosingParen := false
	var invalidArgs []l.Token
	endToken := startToken
	if t := p.expect([]l.TokenKind{l.SymbolRightParen}); t == nil {
		missingClosingParen = true
		invalidArgs = p.recoverFromError(
			diag.UnexpectedToken,
			p.Pos,
			[]l.TokenKind{l.SymbolRightParen, l.NewLine, l.EOF},
		)
		if p.currentToken().Kind == l.SymbolRightParen {
			endToken = p.currentToken()
			_ = p.advance()
		}
	} else {
		endToken = *t
		_ = p.advance()
	}

	return Expression{
		Kind: CallExpression{
			Identifier:          identifier,
			Parameters:          parameters,
			InvalidArgs:         invalidArgs,
			MissingClosingParen: missingClosingParen,
		},
		Range: source.MergeRange(identifier.Range, endToken.Range),
	}
}

func (p *parser) parseCallStatement(
	identifier IdentifierExpression,
	leadingComments []string,
) Statement {
	expr := p.parseCallExpression(identifier)
	var callEpr CallExpression
	if c, ok := expr.Kind.(CallExpression); ok {
		callEpr = c
	}

	invalidAfterArgs, trailingComment := p.parseEndComment()

	return Statement{
		Kind:            CallStatement(callEpr),
		LeadingComments: leadingComments,
		TrailingComment: trailingComment,
		InvalidAfter:    invalidAfterArgs,
		Range:           expr.Range,
	}
}

func tokenToAssignmentKind(token l.Token) *AssignmentKind {
	var assignmentKind AssignmentKind
	switch token.Kind {
	case l.OperatorAssign:
		assignmentKind = Assign
	case l.OperatorAddAssign:
		assignmentKind = AddAssign
	case l.OperatorSubtractAssign:
		assignmentKind = SubtractAssign
	case l.OperatorMultiplyAssign:
		assignmentKind = MultiplyAssign
	case l.OperatorDivideAssign:
		assignmentKind = DivideAssign
	case l.OperatorAndAssign:
		assignmentKind = AndAssign
	case l.OperatorOrAssign:
		assignmentKind = OrAssign
	case l.OperatorAssignIfBlank:
		assignmentKind = AssignIfBlank
	case l.OperatorAssignRaw:
		assignmentKind = AssignRaw
	default:
		return nil
	}
	return &assignmentKind
}

func (p *parser) parseSection() SectionSwitchKind {
	startToken := p.currentToken()
	if startToken.Kind != l.KeywordNamespaceNc && startToken.Kind != l.KeywordNamespaceChan &&
		startToken.Kind != l.KeywordNamespacePs &&
		startToken.Kind != l.KeywordNamespaceBd && startToken.Kind != l.Section {
		panic("must be called with section token")
	}
	startPos := p.Pos

	var ns SectionNamespaceKind
	switch startToken.Kind {
	case l.KeywordNamespaceChan:
		ns = Chandata
		_ = p.advance()
	case l.KeywordNamespaceNc:
		ns = Nc
		_ = p.advance()
	case l.KeywordNamespacePs:
		ns = Ps
		_ = p.advance()
	case l.KeywordNamespaceBd:
		ns = Bd
		_ = p.advance()
	default:
		ns = Unqualified
	}

	lexeme := p.currentToken().Lexeme
	_ = p.advance()

	upper := strings.ToUpper(lexeme)
	switch {
	case strings.Contains(lexeme, "$("):
		return DynamicSection(lexeme)
	case strings.HasPrefix(upper, "[B"):
		if s, err := parseDriveSection(lexeme, ns); err == nil {
			return s
		} else {
			i := p.recoverFromError(
				diag.SectionInvalidDrive,
				startPos,
				[]l.TokenKind{l.NewLine, l.EOF, l.Comment},
			)
			return InvalidSection(i)
		}
	case strings.HasPrefix(upper, "[C"):
		if s, err := parseChanSection(lexeme, ns); err == nil {
			return s
		} else {
			i := p.recoverFromError(
				diag.SectionInvalidChan, startPos,
				[]l.TokenKind{l.NewLine, l.EOF, l.Comment},
			)
			return InvalidSection(i)
		}
	case strings.HasPrefix(lexeme, "("):
		if s, err := parseChandataSection(lexeme, ns); err == nil {
			return s
		} else {
			i := p.recoverFromError(
				diag.SectionInvalidChan, startPos,
				[]l.TokenKind{l.NewLine, l.EOF, l.Comment},
			)
			return InvalidSection(i)
		}
	case strings.EqualFold(lexeme, "[SL]"):
		return DisplaySection{Name: "SL", Namespace: ns}
	default:
		i := p.recoverFromError(
			diag.SectionFormatUnrecogniced, startPos,
			[]l.TokenKind{l.NewLine, l.EOF, l.Comment},
		)
		return InvalidSection(i)
	}
}

func (p *parser) parseNamespacedStatement(leadingComments []string) Statement {
	startToken := p.currentToken()
	section := p.parseSection()
	if p.currentToken().Kind == l.SymbolDot {
		_ = p.advance()
	}
	if p.currentToken().Kind == l.LiteralIdentifier || p.currentToken().Kind == l.SymbolDollarParen {
		identifier := p.parseIdentifier(&section)
		identifier.Range.Start = startToken.Range.Start
		switch {
		case p.currentToken().Kind == l.SymbolLeftParen:
			return p.parseCallStatement(identifier, leadingComments)
		case p.currentToken().Kind == l.OperatorDelete:
			return p.parseDeleteStatement(identifier, leadingComments)
		case tokenToAssignmentKind(p.currentToken()) != nil:
			return p.parseAssignment(identifier, leadingComments)
		}
	}

	invalidEndTokens, trailingComment := p.parseEndComment()
	return Statement{
		Kind:            SectionSwitch{Kind: section, InvalidEndTokens: invalidEndTokens},
		LeadingComments: leadingComments,
		TrailingComment: trailingComment,
		Range:           source.MergeRange(startToken.Range, p.currentToken().Range),
	}
}

func (p *parser) parseSectionStatement(leadingComments []string) Statement {
	startToken := p.currentToken()
	section := p.parseSection()
	endToken := p.currentToken()

	invalidEndTokens, trailingComment := p.parseEndComment()

	return Statement{
		Kind: SectionSwitch{
			Kind:             section,
			InvalidEndTokens: invalidEndTokens,
		},
		LeadingComments: leadingComments,
		TrailingComment: trailingComment,
		Range:           source.MergeRange(startToken.Range, endToken.Range),
	}
}

func parseChandataSection(section string, ns SectionNamespaceKind) (*ChannelSection, error) {
	// format:(<chan>)
	runes := []rune(section)
	if len(runes) < 3 || runes[0] != '(' || runes[len(runes)-1] != ')' {
		return nil, errors.New("invalid CHANDATA section")
	}
	trimmed := string(runes[1 : len(runes)-1])
	channel, err := strconv.ParseUint(trimmed, 10, 8)
	if err != nil {
		return nil, err
	}
	if channel < 1 || channel > 10 {
		return nil, errors.New("channel number must be between 1 and 10")
	}
	c := ChannelSection{
		Channo:    uint8(channel),
		Namespace: ns,
	}

	return &c, nil
}

func parseChanSection(section string, ns SectionNamespaceKind) (*ChannelSection, error) {
	// format: [C<chan>]
	runes := []rune(section)
	if len(runes) < 4 || runes[0] != '[' || unicode.ToUpper(runes[1]) != 'C' || runes[len(runes)-1] != ']' {
		return nil, errors.New("invalid channel section")
	}
	trimmed := string(runes[2 : len(runes)-1])
	channel, err := strconv.ParseUint(trimmed, 10, 8)
	if err != nil {
		return nil, err
	}
	if channel < 1 || channel > 10 {
		return nil, errors.New("channel number must be between 1 and 10")
	}
	c := ChannelSection{
		Channo:    uint8(channel),
		Namespace: ns,
	}

	return &c, nil
}

func parseDriveSection(section string, ns SectionNamespaceKind) (*DriveSection, error) {
	// format: [B<bus>_S<slave>_PS<do>]
	runes := []rune(section)
	if len(runes) < 2 || runes[0] != '[' || runes[len(runes)-1] != ']' {
		return nil, errors.New("invalid drive section")
	}
	trimmed := string(runes[1 : len(runes)-1])
	parts := strings.Split(trimmed, "_")
	if len(parts) != 3 || len(parts[0]) < 2 || len(parts[1]) < 2 || len(parts[2]) < 3 {
		return nil, errors.New("invalid drive section")
	}

	bus, err := strconv.ParseUint(string([]rune(parts[0])[1:]), 10, 8)
	if err != nil {
		return nil, err
	}

	slave, err := strconv.ParseUint(string([]rune(parts[1])[1:]), 10, 8)
	if err != nil {
		return nil, err
	}

	do, err := strconv.ParseUint(string([]rune(parts[2])[2:]), 10, 8)
	if err != nil {
		return nil, err
	}
	if do < 1 || do > 99 {
		return nil, errors.New("drive object number must be between 1 and 99")
	}

	return &DriveSection{
		Bus:       uint8(bus),
		Slave:     uint8(slave),
		Do:        uint8(do),
		Namespace: ns,
	}, nil
}

func (p *parser) parseWhileBlock(leadingComments []string) Statement {
	startToken := p.currentToken()
	if startToken.Kind != l.KeywordWhile {
		panic("must be called with while token")
	}

	_ = p.advance()

	var condition *Expression = nil
	if p.currentToken().Kind != l.NewLine {
		c := p.parseExpression(0)
		condition = &c
	}

	var invalidAfterExpr []l.Token
	if t := p.expect([]l.TokenKind{l.NewLine, l.EOF, l.Comment}); t == nil {
		invalidAfterExpr = p.recoverFromError(
			diag.UnexpectedToken,
			p.Pos,
			[]l.TokenKind{l.NewLine, l.EOF, l.Comment},
		)
	}

	trailingCommentStart := p.parseOptionalComment()

	if t := p.expectAndAdvance([]l.TokenKind{l.NewLine, l.EOF}); t == nil {
		invalidAfterExpr = p.recoverFromError(
			diag.UnexpectedToken,
			p.Pos,
			[]l.TokenKind{l.NewLine, l.EOF},
		)
		_ = p.advance()
	}

	endToken := p.currentToken()
	var trailingCommentEnd *string
	var invalidAfterEnd []l.Token

	body, comments := p.parseBody([]l.TokenKind{l.KeywordEndWhile}, diag.WhileEndMissing)

	if body != nil {
		endToken = p.advance()

		if t := p.expect([]l.TokenKind{l.NewLine, l.EOF, l.Comment}); t == nil {
			invalidAfterExpr = p.recoverFromError(
				diag.UnexpectedToken,
				p.Pos,
				[]l.TokenKind{l.NewLine, l.EOF, l.Comment},
			)
		}

		trailingCommentEnd = p.parseOptionalComment()

		if t := p.expectAndAdvance([]l.TokenKind{l.NewLine, l.EOF}); t == nil {
			invalidAfterExpr = p.recoverFromError(
				diag.UnexpectedToken,
				p.Pos,
				[]l.TokenKind{l.NewLine, l.EOF},
			)
			_ = p.advance()
		}
	}

	return Statement{
		Kind: WhileBlock{
			Condition:            condition,
			Body:                 body,
			LeadingCommentsEnd:   comments,
			TrailingCommentStart: trailingCommentStart,
			InvalidAfterExpr:     invalidAfterExpr,
		},
		LeadingComments: leadingComments,
		TrailingComment: trailingCommentEnd,
		Range:           source.MergeRange(startToken.Range, endToken.Range),
		InvalidAfter:    invalidAfterEnd,
	}
}

func (p *parser) parseIfBlock(leadingComments []string) Statement {
	startToken := p.currentToken()
	_ = p.advance()

	var condition *Expression = nil
	if p.currentToken().Kind != l.NewLine {
		c := p.parseExpression(0)
		condition = &c
	}

	var invalidAfterExpr []l.Token
	if t := p.expect(
		[]l.TokenKind{l.NewLine, l.EOF, l.Comment},
	); t == nil {
		invalidAfterExpr = p.recoverFromError(diag.UnexpectedToken, p.Pos, []l.TokenKind{
			l.NewLine, l.EOF, l.Comment,
		})
	}

	trailingCommentStart := p.parseOptionalComment()
	if t := p.expectAndAdvance(
		[]l.TokenKind{l.NewLine, l.EOF},
	); t == nil {
		invalidAfterExpr = p.recoverFromError(diag.UnexpectedToken, p.Pos, []l.TokenKind{
			l.NewLine, l.EOF,
		})
		_ = p.advance()
	}
	var trailingCommentEnd *string = nil
	var invalidAfterEnd []l.Token = nil

	var elseIfBranchesList []ElseIf = nil
	var elseBranch *Else = nil
	var invalidAfterElse []l.Token = nil
	var comments []string
	thenBody, thenComments := p.parseBody([]l.TokenKind{
		l.KeywordElse, l.KeywordElseIf, l.KeywordEndIf,
	}, diag.IfThenEndMissing)

	if thenBody != nil {
		token := p.currentToken()
		comments = thenComments

		for token.Kind == l.KeywordElseIf {
			_ = p.advance()
			leadingCommentsElseIf := comments
			var elseIfCondition *Expression = nil
			if p.currentToken().Kind != l.NewLine {
				c := p.parseExpression(0)
				elseIfCondition = &c
			}

			var invalidAfterElseIfExpr []l.Token = nil
			if t := p.expect([]l.TokenKind{l.NewLine, l.EOF, l.Comment}); t == nil {
				invalidAfterElseIfExpr = p.recoverFromError(
					diag.UnexpectedToken,
					p.Pos,
					[]l.TokenKind{l.NewLine, l.EOF, l.Comment},
				)
			}

			trailingCommentElseIf := p.parseOptionalComment()

			if t := p.expectAndAdvance([]l.TokenKind{l.NewLine, l.EOF}); t == nil {
				invalidAfterElseIfExpr = p.recoverFromError(
					diag.UnexpectedToken,
					p.Pos,
					[]l.TokenKind{l.NewLine, l.EOF},
				)
				_ = p.advance()
			}

			var elseIfThen []Statement
			elseIfThen, comments = p.parseBody(
				[]l.TokenKind{l.KeywordElse, l.KeywordElseIf, l.KeywordEndIf},
				diag.IfElseIfEndMissing,
			)

			elseIfBranchesList = append(elseIfBranchesList, ElseIf{
				Condition:        elseIfCondition,
				LeadingComments:  leadingCommentsElseIf,
				TrailingComment:  trailingCommentElseIf,
				ThenBranch:       elseIfThen,
				InvalidAfterExpr: invalidAfterElseIfExpr,
			})

			token = p.currentToken()
			if elseIfThen == nil {
				goto end
			}
		}

		if token.Kind == l.KeywordElse {
			_ = p.advance()

			if t := p.expect([]l.TokenKind{l.NewLine, l.EOF, l.Comment}); t == nil {
				invalidAfterElse = p.recoverFromError(
					diag.UnexpectedToken,
					p.Pos,
					[]l.TokenKind{l.NewLine, l.EOF, l.Comment},
				)
			}

			trailingCommentElse := p.parseOptionalComment()

			if t := p.expectAndAdvance([]l.TokenKind{l.NewLine, l.EOF}); t == nil {
				invalidAfterElse = p.recoverFromError(
					diag.UnexpectedToken,
					p.Pos,
					[]l.TokenKind{l.NewLine, l.EOF},
				)
				_ = p.advance()
			}

			leadingCommentsElse := comments
			var elseBody []Statement
			elseBody, comments = p.parseBody(
				[]l.TokenKind{l.KeywordEndIf},
				diag.IfElseEndMissing,
			)

			elseBranch = &Else{
				LeadingComments: leadingCommentsElse,
				TrailingComment: trailingCommentElse,
				ThenBranch:      elseBody,
			}

			if elseBody == nil {
				goto end
			}
		}

		// consume endif
		_ = p.advance()

		if t := p.expect([]l.TokenKind{l.NewLine, l.EOF, l.Comment}); t == nil {
			invalidAfterEnd = p.recoverFromError(
				diag.UnexpectedToken,
				p.Pos,
				[]l.TokenKind{l.NewLine, l.EOF, l.Comment},
			)
		}

		trailingCommentEnd = p.parseOptionalComment()

		if t := p.expectAndAdvance([]l.TokenKind{l.NewLine, l.EOF}); t == nil {
			invalidAfterEnd = p.recoverFromError(
				diag.UnexpectedToken,
				p.Pos,
				[]l.TokenKind{l.NewLine, l.EOF},
			)
			_ = p.advance()
		}
	}

end:

	return Statement{
		Kind: IfBlock{
			Condition:            condition,
			ThenBranch:           thenBody,
			ElseBranch:           elseBranch,
			ElseIfBranch:         elseIfBranchesList,
			LeadingCommentsEnd:   comments,
			TrailingCommentStart: trailingCommentStart,
			InvalidAfterExpr:     invalidAfterExpr,
			InvalidAfterElse:     invalidAfterElse,
		},
		LeadingComments: leadingComments,
		TrailingComment: trailingCommentEnd,
		Range:           source.MergeRange(startToken.Range, p.Tokens[p.Pos-1].Range),
		InvalidAfter:    invalidAfterEnd,
	}
}

func (p *parser) parseBody(
	endToken []l.TokenKind,
	diagKind diag.DiagnosticKind,
) ([]Statement, []string) {
	var comments []string
	var body []Statement
	bodyList := []Statement{}

	for {
		comments = p.parseCommentBlock()
		token := p.currentToken()
		if token.Kind == l.EOF {
			p.Diagnostics = append(p.Diagnostics, diag.Diagnostic{
				Kind:     diagKind,
				Severity: diag.Error,
				Range:    token.Range,
			})

			return bodyList, comments
		}
		if slices.Contains(endToken, token.Kind) {
			body = bodyList
			break
		}

		bodyList = append(bodyList, p.parseStatement(comments))
	}

	return body, comments
}

func (p *parser) parseOptionalComment() *string {
	token := p.currentToken()
	if token.Kind == l.Comment {
		_ = p.advance()
		return &token.Lexeme
	}
	return nil
}

func (p *parser) recoverFromError(
	diagKind diag.DiagnosticKind,
	startPos int,
	nextValidTokenKind []l.TokenKind,
) []l.Token {
	var token l.Token
	if len(nextValidTokenKind) > 0 {
		token = p.currentToken()
		for !slices.Contains(nextValidTokenKind, token.Kind) {
			token = p.advance()
		}
	} else {
		token = p.advance()
	}

	p.Diagnostics = append(p.Diagnostics, diag.Diagnostic{
		Kind:     diagKind,
		Range:    source.MergeRange(p.Tokens[startPos].Range, p.currentToken().Range),
		Severity: diag.Error,
	})

	return p.Tokens[startPos:p.Pos]
}

func (p *parser) parseExpression(minPrec uint8) Expression {
	token := p.currentToken()
	var expression Expression

	switch token.Kind {
	case l.SymbolLeftParen:
		_ = p.advance()
		innerExpression := p.parseExpression(0)

		missingClosingParentheses := false
		endToken := p.currentToken()
		if t := p.expectAndAdvance([]l.TokenKind{l.SymbolRightParen}); t != nil {
			endToken = *t
		} else {
			missingClosingParentheses = true
			p.Diagnostics = append(p.Diagnostics, diag.Diagnostic{
				Kind:     diag.ExpressionGroupedMissingClosingParentheses,
				Range:    source.MergeRange(token.Range, endToken.Range),
				Severity: diag.Error,
			})
		}

		expression.Kind = GroupedExpression{
			Expression:          innerExpression,
			MissingClosingParen: missingClosingParentheses,
		}
		expression.Range = source.MergeRange(token.Range, endToken.Range)
	case l.OperatorNegate, l.OperatorAdd, l.OperatorSubtract:
		expression = p.parsePrefixedExpression()
	case l.LiteralIdentifier, l.SymbolDollarParen:
		identifier := p.parseIdentifier(nil)

		if p.currentToken().Kind == l.SymbolLeftParen {
			expression = p.parseCallExpression(identifier)
		} else if p.currentToken().Kind == l.Unknown && p.currentToken().Lexeme == ":" &&
			p.peekToken(1).Kind == l.LiteralNumber {
			// SINAMICS BICO literals can use a dynamic drive-object number,
			// for example $(Up.doNr):4105.0.
			_ = p.advance()
			tail := p.currentToken()
			_ = p.advance()
			expression.Kind = DynamicLiteral(IdentifierString(identifier) + ":" + tail.Lexeme)
			expression.Range = source.MergeRange(identifier.Range, tail.Range)
		} else {
			expression.Kind = identifier
			expression.Range = identifier.Range
		}
	case l.LiteralString:
		_ = p.advance()
		expression.Kind = parseStringLiteral(token)
		expression.Range = token.Range
	case l.LiteralNumber:
		expression = p.parseNumberLiteral()
	case l.LiteralHex:
		_ = p.advance()
		if strings.Contains(token.Lexeme, "$(") {
			expression = Expression{Kind: DynamicLiteral(token.Lexeme), Range: token.Range}
		} else if len(token.Lexeme) <= 2 {
			p.Diagnostics = append(p.Diagnostics, diag.Diagnostic{Kind: diag.ExpressionInvalid, Range: token.Range, Severity: diag.Error})
			expression = Expression{Kind: Invalid{}, Range: token.Range}
		} else {
			expression = Expression{Kind: HexLiteral(token.Lexeme), Range: token.Range}
		}
	case l.LiteralNumberFormat:
		expression = p.parseFormatNumber()
	case l.LiteralNull:
		_ = p.advance()
		expression = Expression{
			Kind:  NullLiteral{},
			Range: token.Range,
		}
	case l.LiteralTrue:
		_ = p.advance()
		expression = Expression{
			Kind:  TrueLiteral{},
			Range: token.Range,
		}
	case l.LiteralFalse:
		_ = p.advance()
		expression = Expression{
			Kind:  FalseLiteral{},
			Range: token.Range,
		}
	case l.KeywordNamespaceNc:
		fallthrough
	case l.KeywordNamespacePs, l.KeywordNamespaceBd:
		section := p.parseSection()
		if p.currentToken().Kind == l.SymbolDot {
			_ = p.advance()
			ident := p.parseIdentifier(&section)
			ident.Range.Start = token.Range.Start
			expression.Kind = ident
			expression.Range = ident.Range
		} else {
			_ = p.advance()
			p.Diagnostics = append(p.Diagnostics, diag.Diagnostic{
				Kind:     diag.FullyQualifiedIdentMissing,
				Range:    token.Range,
				Severity: diag.Error,
			})

		}
	default:
		_ = p.advance()
		p.Diagnostics = append(p.Diagnostics, diag.Diagnostic{
			Kind:     diag.ExpressionInvalid,
			Range:    token.Range,
			Severity: diag.Error,
		})

		expression = Expression{
			Kind:  Invalid{},
			Range: token.Range,
		}
	}

	for {
		t := p.currentToken()

		op := tokenToBinaryOperator(t)

		if op == nil {
			break
		}

		prec := getPrecedence(*op)

		if prec < minPrec {
			break
		}

		_ = p.advance()

		rhs := p.parseExpression(prec + 1)

		binExpr := Expression{
			Kind: BinaryExpression{
				Op:    *op,
				Left:  expression,
				Right: rhs,
			},
			Range: source.SourceRange{
				Start: expression.Range.Start,
				End:   rhs.Range.End,
			},
		}

		expression = binExpr
	}

	return expression
}

func parseStringLiteral(token l.Token) ExpressionKind {
	replacements := l.StringReplacements(token)
	if len(replacements) == 0 {
		return StringLiteral(token.Lexeme)
	}
	result := InterpolatedStringLiteral{Value: token.Lexeme}
	for _, replacement := range replacements {
		nested := parser{Tokens: replacement.Tokens}
		expression := nested.parseExpression(0)
		identifier, ok := expression.Kind.(IdentifierExpression)
		if ok && nested.currentToken().Kind == l.EOF {
			result.Replacements = append(result.Replacements, identifier)
		}
	}
	return result
}

func getPrecedence(op BinaryOperator) uint8 {
	var prec uint8 = 0

	switch op {
	case Multiply, Divide:
		prec = 7
	case Add, Subtract:
		prec = 6
	case StringConcat:
		prec = 5
	case And:
		prec = 4
	case Or:
		prec = 3
	case Equal, NotEqual, LessThan, LessThanEqual, GreaterThan, GreaterThanEqual:
		prec = 2
	case LogicalAnd:
		prec = 1
	case LogicalOr:
		prec = 0
	}

	return prec
}

func tokenToBinaryOperator(token l.Token) *BinaryOperator {
	var op BinaryOperator

	switch token.Kind {
	case l.OperatorEqual:
		op = Equal
	case l.OperatorUnequal:
		op = NotEqual
	case l.OperatorLessThan:
		op = LessThan
	case l.OperatorLessThanEqual:
		op = LessThanEqual
	case l.OperatorGreaterThan:
		op = GreaterThan
	case l.OperatorGreaterThanEqual:
		op = GreaterThanEqual
	case l.OperatorStringConcat:
		op = StringConcat
	case l.OperatorAdd:
		op = Add
	case l.OperatorSubtract:
		op = Subtract
	case l.OperatorMultiply:
		op = Multiply
	case l.OperatorDivide:
		op = Divide
	case l.OperatorOr:
		op = Or
	case l.OperatorAnd:
		op = And
	case l.OperatorLogOr:
		op = LogicalOr
	case l.OperatorLogAnd:
		op = LogicalAnd
	default:
		return nil
	}

	return &op
}

func (p *parser) parseFormatNumber() Expression {
	token := p.currentToken()
	if token.Kind != l.LiteralNumberFormat {
		panic("asdf")
	}

	_ = p.advance()

	lexLen := len([]rune(token.Lexeme))
	if lexLen < 4 {
		p.Diagnostics = append(p.Diagnostics, diag.Diagnostic{
			Kind:     diag.ExpressionInvalid,
			Range:    token.Range,
			Severity: diag.Error,
		})

		return Expression{
			Kind:  Invalid{},
			Range: token.Range,
		}
	}

	numString := string([]rune(token.Lexeme)[2 : lexLen-1])
	format := unicode.ToLower([]rune(token.Lexeme)[1])
	if strings.Contains(numString, "$(") {
		return Expression{Kind: DynamicLiteral(token.Lexeme), Range: token.Range}
	}

	switch format {
	case 'h':
		if num, err := strconv.ParseUint(numString, 16, 64); err == nil {
			return Expression{
				Kind: BitwiseLiteral{
					Kind:  Hex,
					Value: num,
				},
				Range: token.Range,
			}
		}

		p.Diagnostics = append(p.Diagnostics, diag.Diagnostic{
			Kind:     diag.BitwiseLiteralInvalidHex,
			Range:    token.Range,
			Severity: diag.Error,
		})
	case 'b':
		if num, err := strconv.ParseUint(numString, 2, 64); err == nil {
			return Expression{
				Kind: BitwiseLiteral{
					Kind:  Bin,
					Value: num,
				},
				Range: token.Range,
			}
		}

		p.Diagnostics = append(p.Diagnostics, diag.Diagnostic{
			Kind:     diag.BitwiseLiteralInvalidBin,
			Range:    token.Range,
			Severity: diag.Error,
		})
	default:
		// CMC also uses single-quoted character literals such as '\r' and
		// '\n'. Only B/H-prefixed values have numeric semantics.
		return Expression{Kind: StringLiteral(token.Lexeme), Range: token.Range}
	}

	return Expression{
		Kind:  Invalid{},
		Range: token.Range,
	}
}

func (p *parser) parseNumberLiteral() Expression {
	token := p.currentToken()
	if token.Kind != l.LiteralNumber {
		panic("must be called with number token")
	}

	_ = p.advance()

	dotCount := strings.Count(token.Lexeme, ".")
	colonCount := strings.Count(token.Lexeme, ":")

	switch {
	case (dotCount == 1 && colonCount == 0) ||
		(dotCount <= 1 && colonCount == 0 && p.currentToken().Kind == l.LiteralNumberEx):
		// float number
		if num, err := strconv.ParseFloat(token.Lexeme, 64); err == nil {
			endToken := token
			var exponent *Expression = nil
			if p.currentToken().Kind == l.LiteralNumberEx {
				_ = p.advance()
				e := p.parseExpression(0)
				exponent = &e
				if !isValidExponent(exponent.Kind) {
					p.Diagnostics = append(p.Diagnostics, diag.Diagnostic{
						Kind:     diag.NumberLiteralInvalidExponent,
						Range:    source.MergeRange(token.Range, endToken.Range),
						Severity: diag.Error,
					})
				}
			}
			return Expression{
				Kind: FloatLiteral{
					base:     num,
					exponent: exponent,
				},
				Range: source.MergeRange(token.Range, endToken.Range),
			}
		} else {
			p.Diagnostics = append(p.Diagnostics, diag.Diagnostic{
				Kind:     diag.NumberLiteralParseError,
				Range:    token.Range,
				Severity: diag.Error,
			})
		}
	case dotCount == 0 && colonCount == 0:
		// integer
		if num, err := strconv.ParseInt(token.Lexeme, 10, 64); err == nil {
			return Expression{
				Kind:  IntegerLiteral(num),
				Range: token.Range,
			}
		} else {
			p.Diagnostics = append(p.Diagnostics, diag.Diagnostic{
				Kind:     diag.NumberLiteralParseError,
				Range:    token.Range,
				Severity: diag.Error,
			})
		}
	case dotCount == 1 && colonCount == 1:
		// bico
		if bico, err := parseBico(token.Lexeme); err == nil {
			return Expression{
				Kind:  *bico,
				Range: token.Range,
			}
		}
		p.Diagnostics = append(p.Diagnostics, diag.Diagnostic{
			Kind:     diag.BicoLiteralParseError,
			Range:    token.Range,
			Severity: diag.Error,
		})
	case dotCount > 1 && colonCount == 0:
		// version
		if version, err := parseVersion(token.Lexeme); err == nil {
			return Expression{
				Kind:  *version,
				Range: token.Range,
			}
		}
		p.Diagnostics = append(p.Diagnostics, diag.Diagnostic{
			Kind:     diag.VersionLiteralParserError,
			Range:    token.Range,
			Severity: diag.Error,
		})
	default:
		// invalid
		p.Diagnostics = append(p.Diagnostics, diag.Diagnostic{
			Kind:     diag.ExpressionInvalid,
			Range:    token.Range,
			Severity: diag.Error,
		})
	}

	return Expression{
		Kind:  Invalid{},
		Range: token.Range,
	}
}

func isValidExponent(expression ExpressionKind) bool {
	switch expression := expression.(type) {
	case IdentifierExpression, IntegerLiteral, FloatLiteral:
		return true
	case PrefixedExpression:
		return isValidExponent(expression.Expression.Kind)
	default:
		return false
	}
}

func parseVersion(version string) (*VersionLiteral, error) {
	var versionParts VersionLiteral
	parts := strings.SplitSeq(version, ".")
	for p := range parts {
		num, err := strconv.ParseUint(p, 10, 16)
		if err != nil {
			return nil, err
		}

		versionParts = append(versionParts, uint16(num))
	}

	return &versionParts, nil
}

func parseBico(bico string) (*BicoLiteral, error) {
	if strings.Index(bico, ":") > strings.Index(bico, ".") {
		return nil, errors.New("")
	}

	parts := strings.FieldsFunc(bico, func(r rune) bool { return r == ':' || r == '.' })
	do, err := strconv.ParseUint(parts[0], 10, 16)
	if err != nil {
		return nil, err
	}

	parameter, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return nil, err
	}

	index, err := strconv.ParseUint(parts[2], 10, 16)
	if err != nil {
		return nil, err
	}

	return &BicoLiteral{
		do:        uint16(do),
		parameter: uint16(parameter),
		index:     uint16(index),
	}, nil
}

func (p *parser) parseIdentifier(section *SectionSwitchKind) IdentifierExpression {
	startToken := p.currentToken()
	endToken := startToken
	var segments []IdentifierSegment
	var parts []IdentifierPart

	flush := func() {
		segments = append(segments, IdentifierSegment{Parts: append([]IdentifierPart(nil), parts...)})
		parts = nil
	}

	for {
		token := p.currentToken()
		switch token.Kind {
		case l.LiteralIdentifier:
			parts = append(parts, LiteralIdentifier(token.Lexeme))
			endToken = token
			_ = p.advance()
		case l.SymbolDollarParen:
			replacementStart := token
			_ = p.advance()
			replacement := p.parseIdentifier(nil)
			parts = append(parts, ReplacementIdentifier(replacement))
			endToken = p.Tokens[p.Pos-1]
			if p.currentToken().Kind == l.SymbolRightParen {
				endToken = p.currentToken()
				_ = p.advance()
			} else {
				p.Diagnostics = append(p.Diagnostics, diag.Diagnostic{
					Kind:     diag.ExpressionReplacementMissingClosingParentheses,
					Range:    source.MergeRange(replacementStart.Range, p.currentToken().Range),
					Severity: diag.Error,
				})
			}
		case l.SymbolLeftBracket:
			var raw strings.Builder
			depth := 0
			for {
				t := p.currentToken()
				if t.Kind == l.EOF || t.Kind == l.NewLine {
					break
				}
				raw.WriteString(t.Lexeme)
				endToken = t
				_ = p.advance()
				if t.Kind == l.SymbolLeftBracket {
					depth++
				} else if t.Kind == l.SymbolRightBracket {
					depth--
					if depth == 0 {
						break
					}
				}
			}
			parts = append(parts, IndexIdentifier(raw.String()))
		case l.SymbolDot:
			endToken = token
			flush()
			_ = p.advance()
		default:
			flush()
			return IdentifierExpression{
				Segments: segments,
				Range:    source.MergeRange(startToken.Range, endToken.Range),
				Section:  section,
			}
		}
	}
}

func (p *parser) parsePrefixedExpression() Expression {
	token := p.currentToken()
	var prefixOperator PrefixOperator
	switch token.Kind {
	case l.OperatorAdd:
		prefixOperator = Plus
	case l.OperatorSubtract:
		prefixOperator = Minus
	case l.OperatorNegate:
		prefixOperator = Negate
	default:
		panic("")
	}
	_ = p.advance()
	innerExpression := p.parseExpression(0)
	return Expression{
		Kind: PrefixedExpression{
			Operator:   prefixOperator,
			Expression: innerExpression,
		},
		Range: source.MergeRange(token.Range, innerExpression.Range),
	}
}

func (p *parser) parseCommentBlock() []string {
	var comments []string
	token := p.currentToken()
	for token.Kind == l.Comment {
		comments = append(comments, token.Lexeme)
		_ = p.advance()

		token = p.advance()
	}

	return comments
}
