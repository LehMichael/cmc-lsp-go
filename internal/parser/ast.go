package parser

import (
	"encoding/json"
	"fmt"

	"github.com/lehmichael/cmc-lsp-go/internal/lexer"
	"github.com/lehmichael/cmc-lsp-go/internal/source"
)

type Ast []Statement

type StatementKind interface {
	isStatementKind()
}

type Statement struct {
	Kind            StatementKind
	Range           source.SourceRange
	LeadingComments []string      `json:",omitempty"`
	TrailingComment *string       `json:",omitempty"`
	InvalidAfter    []lexer.Token `json:",omitempty"`
}

func (s Statement) MarshalJSON() ([]byte, error) {
	type alias Statement // alias drops Statement's methods → no recursion
	return json.Marshal(struct {
		Type string `json:"Type"`
		alias
	}{
		Type:  fmt.Sprintf("%T", s.Kind),
		alias: alias(s),
	})
}

//go:generate go tool enumer -type=AssignmentKind -json
type AssignmentKind int

const (
	Assign AssignmentKind = iota
	AddAssign
	SubtractAssign
	MultiplyAssign
	DivideAssign
	AndAssign
	OrAssign
	AssignIfBlank
)

type Assignment struct {
	Kind   AssignmentKind
	Target IdentifierExpression
	Value  Expression
}

func (Assignment) isStatementKind() {}

type IfBlock struct {
	Condition            *Expression
	ThenBranch           []Statement
	ElseBranch           *Else
	ElseIfBranch         []ElseIf
	LeadingCommentsEnd   []string
	TrailingCommentStart *string
	InvalidAfterExpr     []lexer.Token
	InvalidAfterElse     []lexer.Token
}

func (IfBlock) isStatementKind() {}

type Else struct {
	ThenBranch      []Statement
	LeadingComments []string `json:",omitempty"`
	TrailingComment *string  `json:",omitempty"`
}

type ElseIf struct {
	Condition        *Expression
	ThenBranch       []Statement
	LeadingComments  []string      `json:",omitempty"`
	TrailingComment  *string       `json:",omitempty"`
	InvalidAfterExpr []lexer.Token `json:",omitempty"`
}

type WhileBlock struct {
	Condition            *Expression
	Body                 []Statement
	LeadingCommentsEnd   []string      `json:",omitempty"`
	TrailingCommentStart *string       `json:",omitempty"`
	InvalidAfterExpr     []lexer.Token `json:",omitempty"`
}

func (WhileBlock) isStatementKind() {}

type Call Expression

func (Call) isStatementKind() {}

type SectionSwitch struct {
	Kind             SectionSwitchKind
	InvalidEndTokens []lexer.Token `json:",omitempty"`
}

func (SectionSwitch) isStatementKind() {}

type SectionSwitchKind interface {
	isSectionSwitchKind()
}

type ChannelSection uint8

func (ChannelSection) isSectionSwitchKind() {}

type DriveSection struct {
	Bus   uint8
	Slave uint8
	Do    uint8
}

func (DriveSection) isSectionSwitchKind() {}

type InvalidSection []lexer.Token

func (InvalidSection) isSectionSwitchKind() {}

type DeleteStatement struct {
	Identifier IdentifierExpression
}

func (DeleteStatement) isStatementKind() {}

type PreprocessorStatement struct {
	kind            PreprocessorStatementKind
	LeadingComments []string `json:",omitempty"`
	TrailingComment *string  `json:",omitempty"`
}

func (PreprocessorStatement) isStatementKind() {}

type PreprocessorStatementKind interface {
	isPreprocessorStatementKind()
}

type IncludePpStatement struct {
	Path         string
	InvalidToken *lexer.Token `json:",omitempty"`
}

func (IncludePpStatement) isPreprocessorStatementKind() {}

type NewLine struct{}

func (NewLine) isStatementKind() {}

type FunctionStatement struct {
	Kind                     FunctionKind
	Identifier               *IdentifierExpression
	LeadingCommentsEnd       []string `json:",omitempty"`
	TrailingCommentStart     *string  `json:",omitempty"`
	ArgCount                 int
	Body                     []Statement
	InvalidIdentTokens       []lexer.Token `json:",omitempty"`
	InvalidParamTokens       []lexer.Token `json:",omitempty"`
	InvalidAfterOpeningBrace []lexer.Token `json:",omitempty"`
	InvalidOpeningBrace      []lexer.Token `json:",omitempty"`
}

func (FunctionStatement) isStatementKind() {}

type InvalidStatement []lexer.Token

func (InvalidStatement) isStatementKind() {}

//go:generate go tool enumer -type=FunctionKind -json
type FunctionKind int

const (
	Function FunctionKind = iota
	Procedure
)

// Expressions

type ExpressionKind interface {
	isExpressionKind()
}

type IdentifierExpression struct {
	Segments []IdentifierSegment
	Range    source.SourceRange
}

func (IdentifierExpression) isExpressionKind() {}

type IdentifierSegment struct {
	Parts []IdentifierPart
}

type IdentifierPart interface {
	isIdentifierPart()
}

type LiteralIdentifier string

func (LiteralIdentifier) isIdentifierPart() {}

func (l LiteralIdentifier) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type  string
		Value string
	}{
		Type:  fmt.Sprintf("%T", l),
		Value: string(l),
	})
}

type ReplacementIdentifier IdentifierExpression

func (ReplacementIdentifier) isIdentifierPart() {}

func (r ReplacementIdentifier) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type  string
		Value IdentifierExpression
	}{
		Type:  fmt.Sprintf("%T", r),
		Value: IdentifierExpression(r),
	})
}

type StringLiteral string

func (StringLiteral) isExpressionKind() {}

type IntegerLiteral int64

func (IntegerLiteral) isExpressionKind() {}

type FloatLiteral struct {
	base     float64
	exponent *Expression
}

func (FloatLiteral) isExpressionKind() {}

//go:generate go tool enumer -type=BitwiseLiteralKind -json
type BitwiseLiteralKind int

const (
	Bin BitwiseLiteralKind = iota
	Hex
)

type BitwiseLiteral struct {
	Kind  BitwiseLiteralKind
	Value uint64
}

func (BitwiseLiteral) isExpressionKind() {}

type VersionLiteral []uint16

func (VersionLiteral) isExpressionKind() {}

type BicoLiteral struct {
	do        uint16
	parameter uint16
	index     uint16
}

func (BicoLiteral) isExpressionKind() {}

//go:generate go tool enumer -type=BinaryOperator -json
type BinaryOperator int

const (
	Equal BinaryOperator = iota
	NotEqual
	LessThan
	LessThanEqual
	GreaterThan
	GreaterThanEqual
	StringConcat
	Add
	Subtract
	Multiply
	Divide
	Or
	And
	LogicalOr
	LogicalAnd
)

type BinaryExpression struct {
	Op    BinaryOperator
	Left  Expression
	Right Expression
}

func (BinaryExpression) isExpressionKind() {}

type GroupedExpression struct {
	Expression          Expression
	MissingClosingParen bool
}

func (GroupedExpression) isExpressionKind() {}

//go:generate go tool enumer -type=PrefixOperator -json
type PrefixOperator int

const (
	Plus PrefixOperator = iota
	Minus
	Negate
)

type PrefixedExpression struct {
	Operator   PrefixOperator
	Expression Expression
}

func (PrefixedExpression) isExpressionKind() {}

type CallExpression struct {
	Identifier          IdentifierExpression
	Parameters          []Expression
	InvalidArgs         []lexer.Token
	InvalidAfterArgs    []lexer.Token
	MissingClosingParen bool
	LeadingComments     []string `json:",omitempty"`
	TrailingComment     *string  `json:",omitempty"`
}

func (CallExpression) isExpressionKind() {}

type NullLiteral struct{}

func (NullLiteral) isExpressionKind() {}

type TrueLiteral struct{}

func (TrueLiteral) isExpressionKind() {}

type FalseLiteral struct{}

func (FalseLiteral) isExpressionKind() {}

type Invalid struct{}

func (Invalid) isExpressionKind() {}

type Expression struct {
	Kind  ExpressionKind
	Range source.SourceRange
}

func (e Expression) MarshalJSON() ([]byte, error) {
	type alias Expression
	return json.Marshal(struct {
		Type string `json:"Type"`
		alias
	}{
		Type:  fmt.Sprintf("%T", e.Kind),
		alias: alias(e),
	})
}
