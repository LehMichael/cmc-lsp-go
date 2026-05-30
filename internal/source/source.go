package source

type SourceLocation struct {
	Line   int
	Column int
}

type SourceRange struct {
	Start SourceLocation
	End   SourceLocation
}

func NewRange(line, col, len int) SourceRange {
	start := SourceLocation{line, col}
	end := SourceLocation{line, col + len}
	return SourceRange{start, end}
}

func MergeRange(start, end SourceRange) SourceRange {
	return SourceRange{start.Start, end.End}
}

// func FromToken(token lexer.Token) SourceRange {
// 	return SourceRange{
// 		Start: token.Location,
// 		End:   LocatonFromTokenEnd(token),
// 	}
// }
//
// func FromTokenRange(start lexer.Token, end lexer.Token) SourceRange {
// 	return SourceRange{
// 		Start: start.Location,
// 		End:   LocatonFromTokenEnd(end),
// 	}
// }
//
// func LocatonFromTokenEnd(token lexer.Token) SourceLocation {
// 	return SourceLocation{
// 		Line:   token.Location.Line,
// 		Column: token.Location.Column + len(token.Lexeme),
// 	}
// }
