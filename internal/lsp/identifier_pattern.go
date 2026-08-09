package lsp

import (
	"unicode"

	"github.com/lehmichael/cmc-lsp-go/internal/parser"
)

type identifierPattern []segmentPattern

type segmentPattern []patternToken

type patternToken struct {
	literal  rune
	wildcard bool
}

func identifierPatternFromAST(identifier parser.IdentifierExpression) identifierPattern {
	result := make(identifierPattern, 0, len(identifier.Segments)+1)
	if identifier.Section != nil {
		result = append(result, patternFromRaw(parser.SectionString(*identifier.Section)))
	}
	for _, segment := range identifier.Segments {
		var pattern segmentPattern
		for _, part := range segment.Parts {
			switch part := part.(type) {
			case parser.LiteralIdentifier:
				pattern = appendLiteral(pattern, string(part))
			case parser.ReplacementIdentifier:
				pattern = appendWildcard(pattern)
			case parser.IndexIdentifier:
				pattern = append(pattern, patternFromRaw(string(part))...)
			}
		}
		result = append(result, pattern)
	}
	return result
}

// patternFromRaw turns replacements embedded in raw section/index syntax into
// wildcards while keeping the surrounding literal syntax significant.
func patternFromRaw(value string) segmentPattern {
	runes := []rune(value)
	var result segmentPattern
	for index := 0; index < len(runes); {
		if runes[index] != '$' || index+1 >= len(runes) || runes[index+1] != '(' {
			result = append(result, patternToken{literal: unicode.ToLower(runes[index])})
			index++
			continue
		}
		result = appendWildcard(result)
		index += 2
		depth := 1
		for index < len(runes) && depth > 0 {
			switch runes[index] {
			case '(':
				depth++
			case ')':
				depth--
			}
			index++
		}
	}
	return result
}

func appendLiteral(pattern segmentPattern, value string) segmentPattern {
	for _, character := range value {
		pattern = append(pattern, patternToken{literal: unicode.ToLower(character)})
	}
	return pattern
}

func appendWildcard(pattern segmentPattern) segmentPattern {
	if len(pattern) == 0 || !pattern[len(pattern)-1].wildcard {
		pattern = append(pattern, patternToken{wildcard: true})
	}
	return pattern
}

func identifierPatternsOverlap(left, right identifierPattern) bool {
	if len(left) == 0 || len(left) != len(right) {
		return false
	}
	for index := range left {
		if !segmentPatternsOverlap(left[index], right[index]) {
			return false
		}
	}
	return true
}

// segmentPatternsOverlap checks whether the intersection of two glob-like
// patterns is non-empty. Each wildcard is an NFA state that accepts any number
// of characters, so a small product-state search handles wildcard-vs-wildcard
// cases without guessing concrete replacement values.
func segmentPatternsOverlap(left, right segmentPattern) bool {
	type state struct{ left, right int }
	queue := []state{{}}
	seen := map[state]struct{}{}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if _, ok := seen[current]; ok {
			continue
		}
		seen[current] = struct{}{}
		if current.left == len(left) && current.right == len(right) {
			return true
		}

		leftWildcard := current.left < len(left) && left[current.left].wildcard
		rightWildcard := current.right < len(right) && right[current.right].wildcard
		if leftWildcard {
			queue = append(queue, state{left: current.left + 1, right: current.right})
		}
		if rightWildcard {
			queue = append(queue, state{left: current.left, right: current.right + 1})
		}
		if current.left >= len(left) || current.right >= len(right) {
			continue
		}
		if !leftWildcard && !rightWildcard && left[current.left].literal != right[current.right].literal {
			continue
		}
		nextLeft := current.left + 1
		if leftWildcard {
			nextLeft = current.left
		}
		nextRight := current.right + 1
		if rightWildcard {
			nextRight = current.right
		}
		queue = append(queue, state{left: nextLeft, right: nextRight})
	}
	return false
}
