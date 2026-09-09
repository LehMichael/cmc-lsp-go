package lexer_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/lehmichael/cmc-lsp-go/internal/lexer"
)

func TestLexer_NextToken(t *testing.T) {
	tests := []struct {
		name    string // description of this test case
		input   string
		want    []lexer.Token
		wantErr bool
	}{
		{
			"identifier",
			"myVar",
			[]lexer.Token{{Kind: lexer.LiteralIdentifier, Lexeme: "myVar"}},
			false,
		},

		{
			name:  "simple identifier",
			input: "myVar",
			want: []lexer.Token{
				{
					Kind:   lexer.LiteralIdentifier,
					Lexeme: "myVar",
				},
			},
			wantErr: false,
		},
		{
			name:  "basic assignment",
			input: "x =  42\n",
			want: []lexer.Token{
				{
					Kind:   lexer.LiteralIdentifier,
					Lexeme: "x",
				},
				{
					Kind:   lexer.OperatorAssign,
					Lexeme: "=",
				},
				{
					Kind:   lexer.LiteralNumber,
					Lexeme: "42",
				},
				{
					Kind:   lexer.NewLine,
					Lexeme: "\n",
				},
			},
			wantErr: false,
		},
		{
			name:  "call",
			input: "test(13)",
			want: []lexer.Token{
				{Kind: lexer.LiteralIdentifier, Lexeme: "test"},
				{Kind: lexer.SymbolLeftParen, Lexeme: "("},
				{Kind: lexer.LiteralNumber, Lexeme: "13"},
				{Kind: lexer.SymbolRightParen, Lexeme: ")"},
			},
		},
		{
			name:  "call assign",
			input: "a = test(13)",
			want: []lexer.Token{
				{Kind: lexer.LiteralIdentifier, Lexeme: "a"},
				{Kind: lexer.OperatorAssign, Lexeme: "="},
				{Kind: lexer.LiteralIdentifier, Lexeme: "test"},
				{Kind: lexer.SymbolLeftParen, Lexeme: "("},
				{Kind: lexer.LiteralNumber, Lexeme: "13"},
				{Kind: lexer.SymbolRightParen, Lexeme: ")"},
			},
		},
		{
			name:  "leading N-number annotation",
			input: "  N20000 Up.N20000",
			want: []lexer.Token{
				{Kind: lexer.LiteralBlockNumber, Lexeme: "N20000"},
				{Kind: lexer.LiteralIdentifier, Lexeme: "Up"},
				{Kind: lexer.SymbolDot, Lexeme: "."},
				{Kind: lexer.LiteralIdentifier, Lexeme: "N20000"},
			},
		},
		{
			name:  "section namespaces and member names",
			input: "Up.nc = NC[C1].$MC_CHAN_NAME\nUp.ps = SomeCall(Up.bd, PS[B3_S3_PS3].p105, Up.chandata)",
			want: []lexer.Token{
				{Kind: lexer.LiteralIdentifier, Lexeme: "Up"},
				{Kind: lexer.SymbolDot, Lexeme: "."},
				{Kind: lexer.LiteralIdentifier, Lexeme: "nc"},
				{Kind: lexer.OperatorAssign, Lexeme: "="},
				{Kind: lexer.KeywordNamespaceNc, Lexeme: "NC"},
				{Kind: lexer.Section, Lexeme: "[C1]"},
				{Kind: lexer.SymbolDot, Lexeme: "."},
				{Kind: lexer.LiteralIdentifier, Lexeme: "$MC_CHAN_NAME"},
				{Kind: lexer.NewLine, Lexeme: "\n"},
				{Kind: lexer.LiteralIdentifier, Lexeme: "Up"},
				{Kind: lexer.SymbolDot, Lexeme: "."},
				{Kind: lexer.LiteralIdentifier, Lexeme: "ps"},
				{Kind: lexer.OperatorAssign, Lexeme: "="},
				{Kind: lexer.LiteralIdentifier, Lexeme: "SomeCall"},
				{Kind: lexer.SymbolLeftParen, Lexeme: "("},
				{Kind: lexer.LiteralIdentifier, Lexeme: "Up"},
				{Kind: lexer.SymbolDot, Lexeme: "."},
				{Kind: lexer.LiteralIdentifier, Lexeme: "bd"},
				{Kind: lexer.SymbolComma, Lexeme: ","},
				{Kind: lexer.KeywordNamespacePs, Lexeme: "PS"},
				{Kind: lexer.Section, Lexeme: "[B3_S3_PS3]"},
				{Kind: lexer.SymbolDot, Lexeme: "."},
				{Kind: lexer.LiteralIdentifier, Lexeme: "p105"},
				{Kind: lexer.SymbolComma, Lexeme: ","},
				{Kind: lexer.LiteralIdentifier, Lexeme: "Up"},
				{Kind: lexer.SymbolDot, Lexeme: "."},
				{Kind: lexer.LiteralIdentifier, Lexeme: "chandata"},
				{Kind: lexer.SymbolRightParen, Lexeme: ")"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer(tt.input)

			var got []lexer.Token
			for {
				t := l.NextToken()
				if t.Kind == lexer.EOF {
					break
				}
				got = append(got, t)
			}

			lexCmp := cmpopts.IgnoreFields(lexer.Token{}, "Range", "LeadingWhitespace")
			if diff := cmp.Diff(tt.want, got, lexCmp); diff != "" {
				t.Errorf("tokens mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
