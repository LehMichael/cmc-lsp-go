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
					Kind:              lexer.LiteralIdentifier,
					Lexeme:            "myVar",
					LeadingWhitespace: "",
					Location:          lexer.SourceLocation{Line: 0, Column: 0},
				},
			},
			wantErr: false,
		},
		{
			name:  "basic assignment",
			input: "x =  42\n",
			want: []lexer.Token{
				{
					Kind:              lexer.LiteralIdentifier,
					Lexeme:            "x",
					LeadingWhitespace: "",
					Location:          lexer.SourceLocation{Line: 0, Column: 0},
				},
				{
					Kind:              lexer.OperatorAssign,
					Lexeme:            "=",
					LeadingWhitespace: " ",
					Location:          lexer.SourceLocation{Line: 0, Column: 2},
				},
				{
					Kind:              lexer.LiteralNumber,
					Lexeme:            "42",
					LeadingWhitespace: "  ",
					Location:          lexer.SourceLocation{Line: 0, Column: 5},
				},
				{
					Kind:              lexer.NewLine,
					Lexeme:            "\n",
					LeadingWhitespace: "",
					Location:          lexer.SourceLocation{Line: 0, Column: 7},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer(tt.input)

			var got []lexer.Token
			var gotErr error = nil
			for {
				t, e := l.NextToken()
				if e != nil {
					gotErr = e
					break
				}
				if t.Kind == lexer.EOF {
					break
				}
				got = append(got, *t)
			}

			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("NextToken() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("NextToken() succeeded unexpectedly")
			}

			lexCmp := cmpopts.IgnoreFields(lexer.Token{}, "Location", "LeadingWhitespace")
			if diff := cmp.Diff(tt.want, got, lexCmp); diff != "" {
				t.Errorf("tokens mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
