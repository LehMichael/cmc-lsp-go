package parser

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

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
		token := p.advance()
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
	case l.Section:
		return p.parseSection(leadingComments)
	case l.LiteralIdentifier:
		return p.parseIdentifierStatement(leadingComments)
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
		i := p.parseIdentifier()
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
		token := p.currentToken()
	loop:
		for {
			switch token.Kind {
			case l.SymbolRightParen:
				_ = p.advance()
				break loop
			case l.SymbolDollar:
				arcCount++
				nextToken := p.advance()
				switch nextToken.Kind {
				case l.SymbolComma:
					_ = p.advance()
				case l.SymbolRightParen:
				default:
					invalidParams = true
				}
				token = p.currentToken()
			default:
				invalidParams = true
				break loop
			}
		}
	}

	var invalidParamTokens []l.Token
	if invalidParams {
		invalidParamTokens = p.recoverFromError(
			diag.FunctionInvalidParameterDef,
			paramTokenPos,
			[]l.TokenKind{l.SymbolLeftBrace, l.NewLine, l.Comment, l.EOF},
		)
	}

	var invalidOpeningBraceTokens []l.Token
	if t := p.expectAndAdvance([]l.TokenKind{l.SymbolLeftBrace}); t == nil {
		invalidOpeningBraceTokens = p.recoverFromError(
			diag.FunctionInvalidOpeningBrace,
			paramTokenPos,
			[]l.TokenKind{l.Comment, l.NewLine, l.EOF},
		)
	}

	trailingCommentStart := p.parseOptionalComment()

	var invalidAfterOpeningBraceTokens []l.Token
	if t := p.expectAndAdvance([]l.TokenKind{l.NewLine}); t == nil {
		invalidAfterOpeningBraceTokens = p.recoverFromError(
			diag.UnexpectedToken,
			p.Pos,
			[]l.TokenKind{l.NewLine, l.EOF},
		)
		_ = p.advance()
	}

	var comments []string
	var body []Statement
	endToken := p.currentToken()
	var trailingCommentEnd *string
	var invalidAfterClosingBraceTokens []l.Token

	if invalidOpeningBraceTokens == nil {
		body, comments = p.parseBody(
			[]l.TokenKind{l.SymbolRightBrace},
			diag.FunctionBodyClosingBraceMissing,
		)

		endToken = p.currentToken()
		_ = p.advance()

		invalidAfterClosingBraceTokens, trailingCommentEnd = p.parseEndComment()
	}

	leadingCommentsEnd := comments

	return Statement{
		Kind: FunctionStatement{
			Kind:                     kind,
			Identifier:               identifier,
			ArgCount:                 arcCount,
			Body:                     body,
			TrailingCommentStart:     trailingCommentStart,
			LeadingCommentsEnd:       leadingCommentsEnd,
			InvalidIdentTokens:       invalidIdentTokens,
			InvalidParamTokens:       invalidParamTokens,
			InvalidAfterOpeningBrace: invalidAfterOpeningBraceTokens,
		},
		LeadingComments: leadingComments,
		InvalidAfter:    invalidAfterClosingBraceTokens,
		TrailingComment: trailingCommentEnd,
		Range:           source.MergeRange(startToken.Range, endToken.Range),
	}
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
			tokens := p.recoverFromError(diag.UnexpectedToken, p.Pos, []l.TokenKind{})
			invalidToken = &tokens[0]
			endToken = tokens[0]
		}

		kind = IncludePpStatement{
			Path:         path,
			InvalidToken: invalidToken,
		}
	}

	invalidAfter, trailingComment := p.parseEndComment()

	return Statement{
		Kind: PreprocessorStatement{
			kind: kind,
		},
		LeadingComments: leadingComments,
		TrailingComment: trailingComment,
		InvalidAfter:    invalidAfter,
		Range:           source.MergeRange(startToken.Range, endToken.Range),
	}
}

func (p *parser) parseIdentifierStatement(leadingComments []string) Statement {
	identifier := p.parseIdentifier()

	// ij, _ := json.MarshalIndent(identifier, "", "    ")
	// fmt.Printf("identifier %s\n", string(ij))

	token := p.currentToken()

	isAssignment := tokenToAssignmentKind(token) != nil

	// fmt.Printf("kind: %v, isAssignment %v\n", token.Kind.String(), isAssignment)

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

	value := p.parseExpression(0)

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

func (p *parser) parseCallStatement(
	identifier IdentifierExpression,
	leadingComments []string,
) Statement {
	startToken := p.currentToken()
	if startToken.Kind != l.SymbolLeftParen {
		panic("must be called with SymbolLeftParen token")
	}
	nextToken := p.advance()

	var parameters []Expression

	for {
		parameters = append(parameters, p.parseExpression(0))

		nextToken = p.currentToken()
		if nextToken.Kind == l.SymbolComma {
			nextToken = p.advance()
		} else {
			break
		}
	}

	endToken := p.currentToken()

	missingClosingParen := false
	var invalidArgs []l.Token
	if t := p.expectAndAdvance([]l.TokenKind{l.SymbolRightParen}); t == nil {
		missingClosingParen = true
		invalidArgs = p.recoverFromError(
			diag.UnexpectedToken,
			p.Pos,
			[]l.TokenKind{l.SymbolRightParen, l.NewLine, l.EOF},
		)
		endToken = p.currentToken()
		if p.currentToken().Kind == l.SymbolRightParen {
			_ = p.advance()
		}
	}

	var invalidAfterArgs []l.Token
	if t := p.expect([]l.TokenKind{l.NewLine, l.EOF, l.Comment}); t == nil {
		invalidAfterArgs = p.recoverFromError(
			diag.UnexpectedToken,
			p.Pos,
			[]l.TokenKind{l.NewLine, l.EOF, l.Comment},
		)
	}
	trailingComment := p.parseOptionalComment()
	if t := p.expectAndAdvance([]l.TokenKind{l.NewLine, l.EOF}); t == nil {
		invalidAfterArgs = p.recoverFromError(
			diag.UnexpectedToken,
			p.Pos,
			[]l.TokenKind{l.NewLine, l.EOF},
		)
		_ = p.advance()
	}

	return Statement{
		Kind: Call{
			Kind: CallExpression{
				Identifier:          identifier,
				Parameters:          parameters,
				LeadingComments:     leadingComments,
				TrailingComment:     trailingComment,
				InvalidArgs:         invalidArgs,
				InvalidAfterArgs:    invalidAfterArgs,
				MissingClosingParen: missingClosingParen,
			},
			Range: source.MergeRange(startToken.Range, endToken.Range),
		},
		Range: source.MergeRange(startToken.Range, endToken.Range),
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
	default:
		return nil
	}
	return &assignmentKind
}

func (p *parser) parseSection(leadingComments []string) Statement {
	startToken := p.currentToken()
	if startToken.Kind != l.Section {
		panic("must be called with section token")
	}
	startPos := p.Pos

	_ = p.advance()

	var section SectionSwitchKind
	lexeme := startToken.Lexeme

	switch {
	case strings.HasPrefix(lexeme, "[B"):
		if s, err := parseDriveSection(lexeme); err == nil {
			section = s
		} else {
			i := p.recoverFromError(
				diag.SectionInvalidDrive,
				startPos,
				[]l.TokenKind{l.NewLine, l.EOF, l.Comment},
			)
			section = InvalidSection(i)
		}
	case strings.HasPrefix(lexeme, "[C"):
		if s, err := parseChanSection(lexeme); err == nil {
			section = s
		} else {
			i := p.recoverFromError(
				diag.SectionInvalidChan, startPos,
				[]l.TokenKind{l.NewLine, l.EOF, l.Comment},
			)
			section = InvalidSection(i)
		}
	case strings.HasPrefix(lexeme, "CHANDATA("):
		if s, err := parseChandataSection(lexeme); err == nil {
			section = s
		} else {
			i := p.recoverFromError(
				diag.SectionInvalidChan, startPos,
				[]l.TokenKind{l.NewLine, l.EOF, l.Comment},
			)
			section = InvalidSection(i)
		}
	default:
		i := p.recoverFromError(
			diag.SectionFormatUnrecogniced, startPos,
			[]l.TokenKind{l.NewLine, l.EOF, l.Comment},
		)
		section = InvalidSection(i)
	}

	var invalidEndTokens []l.Token
	if t := p.expect([]l.TokenKind{l.NewLine, l.EOF, l.Comment}); t == nil {
		invalidEndTokens = p.recoverFromError(
			diag.UnexpectedToken,
			p.Pos,
			[]l.TokenKind{l.NewLine, l.EOF, l.Comment},
		)
	}

	trailingComment := p.parseOptionalComment()

	if t := p.expectAndAdvance([]l.TokenKind{l.NewLine, l.EOF}); t == nil {
		invalidEndTokens = p.recoverFromError(
			diag.UnexpectedToken,
			p.Pos,
			[]l.TokenKind{l.NewLine, l.EOF},
		)
		_ = p.advance()
	}

	return Statement{
		Kind: SectionSwitch{
			Kind:             section,
			InvalidEndTokens: invalidEndTokens,
		},
		LeadingComments: leadingComments,
		TrailingComment: trailingComment,
		// Range:           source.FromToken(startToken),
		Range: startToken.Range,
	}
}

func parseChandataSection(section string) (*ChannelSection, error) {
	// format: CHANDATA(<chan>)
	trimmed := string([]rune(section)[9 : utf8.RuneCountInString(section)-1])
	channel, err := strconv.ParseUint(trimmed, 10, 8)
	if err != nil {
		return nil, err
	}
	c := ChannelSection(channel)

	return &c, nil
}

func parseChanSection(section string) (*ChannelSection, error) {
	// format: [C<chan>]
	trimmed := string([]rune(section)[2 : utf8.RuneCountInString(section)-1])
	channel, err := strconv.ParseUint(trimmed, 10, 8)
	if err != nil {
		return nil, err
	}
	c := ChannelSection(channel)

	return &c, nil
}

func parseDriveSection(section string) (*DriveSection, error) {
	// format: [B<bus>_S<slave>_PS<do>]
	trimmed := string([]rune(section)[1 : utf8.RuneCountInString(section)-1])
	parts := strings.Split(trimmed, "_")

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

	return &DriveSection{
		Bus:   uint8(bus),
		Slave: uint8(slave),
		Do:    uint8(do),
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
				[]l.TokenKind{l.KeywordElse, l.KeywordEndIf},
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
	bodyStartPos := p.Pos
	currDiagCount := len(p.Diagnostics)
	var comments []string
	var body []Statement
	var bodyList []Statement

	for {
		comments = p.parseCommentBlock()
		token := p.currentToken()
		if token.Kind == l.EOF {
			// roll back
			p.Pos = bodyStartPos
			p.Diagnostics = p.Diagnostics[:currDiagCount]

			p.Diagnostics = append(p.Diagnostics, diag.Diagnostic{
				Kind:     diagKind,
				Severity: diag.Error,
			})

			return nil, nil
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
		innerIdent := p.parseIdentifier()

		expression.Kind = innerIdent
		expression.Range = innerIdent.Range
	case l.LiteralString:
		_ = p.advance()
		expression.Kind = StringLiteral(token.Lexeme)
		expression.Range = token.Range
	case l.LiteralNumber:
		expression = p.parseNumberLiteral()
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

func getPrecedence(op BinaryOperator) uint8 {
	var prec uint8 = 0

	switch op {
	case Multiply, Divide:
		prec = 4
	case Add, Subtract:
		prec = 3
	case StringConcat:
		prec = 2
	case Equal, NotEqual, LessThan, LessThanEqual, GreaterThan, GreaterThanEqual:
		prec = 1
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

	switch format {
	case 'h':
		if num, err := strconv.ParseUint(numString, 10, 64); err == nil {
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
		if num, err := strconv.ParseUint(numString, 10, 64); err == nil {
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
				switch exponent.Kind.(type) {
				case IdentifierExpression, FloatLiteral:
				default:
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

	parameter, err := strconv.ParseUint(bico, 10, 16)
	if err != nil {
		return nil, err
	}

	index, err := strconv.ParseUint(bico, 10, 16)
	if err != nil {
		return nil, err
	}

	return &BicoLiteral{
		do:        uint16(do),
		parameter: uint16(parameter),
		index:     uint16(index),
	}, nil
}

func (p *parser) parseIdentifier() IdentifierExpression {
	startToken := p.currentToken()
	token := startToken
	endToken := startToken

	var segments []IdentifierSegment
	var parts []IdentifierPart

loop:
	for {
		switch token.Kind {
		case l.LiteralIdentifier:
			parts = append(parts, LiteralIdentifier(token.Lexeme))
		case l.SymbolDollarParen:
			_ = p.advance()
			replacement := p.parseIdentifier()
			parts = append(parts, ReplacementIdentifier(replacement))
			if t := p.expectAndAdvance([]l.TokenKind{l.SymbolRightParen}); t == nil {
				p.Diagnostics = append(p.Diagnostics, diag.Diagnostic{
					Kind:     diag.ExpressionReplacementMissingClosingParentheses,
					Range:    source.MergeRange(startToken.Range, p.currentToken().Range),
					Severity: diag.Error,
				})
			}
		case l.SymbolDot:
			var segment IdentifierSegment
			segment.Parts = append(segment.Parts, parts...)
			segments = append(segments, segment)
			parts = parts[0:0]
		default:
			break loop
		}

		endToken = p.currentToken()
		token = p.advance()
	}

	segments = append(segments, IdentifierSegment{Parts: parts})

	return IdentifierExpression{
		Segments: segments,
		Range:    source.MergeRange(startToken.Range, endToken.Range),
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
