// Package document adapts container formats that embed CMC source.
package document

import (
	"html"
	"path/filepath"
	"strings"

	"github.com/lehmichael/cmc-lsp-go/internal/formatter"
)

type scriptRegion struct {
	start int
	end   int
}

// CMCText returns text suitable for lexing while preserving every original
// line and column. .upact files are XML action documents whose <script>
// elements contain CMC; XML outside those elements is replaced by spaces.
func CMCText(path, text string) string {
	if !strings.EqualFold(filepath.Ext(path), ".upact") {
		return text
	}
	regions := scriptRegions(text)
	if len(regions) == 0 {
		return strings.Map(maskRune, text)
	}
	var result strings.Builder
	result.Grow(len(text))
	cursor := 0
	for _, region := range regions {
		result.WriteString(strings.Map(maskRune, text[cursor:region.start]))
		result.WriteString(unescapePadded(text[region.start:region.end]))
		cursor = region.end
	}
	result.WriteString(strings.Map(maskRune, text[cursor:]))
	return result.String()
}

// SemanticText masks XML entities in addition to the container markup. The
// Tree-sitter grammar highlights those entities; omitting them from semantic
// tokens prevents a decoded operator from being applied to only part of the
// longer XML spelling (for example, &amp;&amp;).
func SemanticText(path, text string) string {
	if !strings.EqualFold(filepath.Ext(path), ".upact") {
		return text
	}
	regions := scriptRegions(text)
	var result strings.Builder
	result.Grow(len(text))
	cursor := 0
	for _, region := range regions {
		result.WriteString(strings.Map(maskRune, text[cursor:region.start]))
		result.WriteString(maskEntities(text[region.start:region.end]))
		cursor = region.end
	}
	result.WriteString(strings.Map(maskRune, text[cursor:]))
	return result.String()
}

// Format applies the CMC formatter only to embedded script elements in an
// .upact document and leaves its XML structure untouched.
func Format(path, text string, options formatter.Options) string {
	if !strings.EqualFold(filepath.Ext(path), ".upact") {
		return formatter.Format(text, options)
	}
	regions := scriptRegions(text)
	if len(regions) == 0 {
		return text
	}
	var result strings.Builder
	cursor := 0
	for _, region := range regions {
		result.WriteString(text[cursor:region.start])
		original := text[region.start:region.end]
		decoded := html.UnescapeString(original)
		formatted := formatter.Format(decoded, options)
		if formatted == decoded {
			result.WriteString(original)
		} else {
			result.WriteString(escapeXMLText(formatted))
		}
		cursor = region.end
	}
	result.WriteString(text[cursor:])
	return result.String()
}

func scriptRegions(text string) []scriptRegion {
	lower := strings.ToLower(text)
	var result []scriptRegion
	cursor := 0
	for cursor < len(text) {
		openOffset := strings.Index(lower[cursor:], "<script")
		if openOffset < 0 {
			break
		}
		open := cursor + openOffset
		tagEndOffset := strings.IndexByte(lower[open:], '>')
		if tagEndOffset < 0 {
			break
		}
		start := open + tagEndOffset + 1
		closeOffset := strings.Index(lower[start:], "</script>")
		if closeOffset < 0 {
			break
		}
		end := start + closeOffset
		result = append(result, scriptRegion{start: start, end: end})
		cursor = end + len("</script>")
	}
	return result
}

func maskRune(value rune) rune {
	if value == '\n' || value == '\r' {
		return value
	}
	return ' '
}

func unescapePadded(text string) string {
	var result strings.Builder
	for index := 0; index < len(text); {
		if text[index] != '&' {
			result.WriteByte(text[index])
			index++
			continue
		}
		start := index
		var decoded strings.Builder
		for index < len(text) && text[index] == '&' {
			endOffset := strings.IndexByte(text[index:], ';')
			if endOffset < 0 {
				break
			}
			end := index + endOffset + 1
			encodedEntity := text[index:end]
			decodedEntity := html.UnescapeString(encodedEntity)
			if decodedEntity == encodedEntity {
				break
			}
			decoded.WriteString(decodedEntity)
			index = end
		}
		if index == start {
			result.WriteByte(text[index])
			index++
			continue
		}
		value := decoded.String()
		result.WriteString(value)
		padding := len([]rune(text[start:index])) - len([]rune(value))
		result.WriteString(strings.Repeat(" ", padding))
	}
	return result.String()
}

func escapeXMLText(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	return strings.ReplaceAll(text, "<", "&lt;")
}

func maskEntities(text string) string {
	var result strings.Builder
	for index := 0; index < len(text); {
		if text[index] != '&' {
			result.WriteByte(text[index])
			index++
			continue
		}
		endOffset := strings.IndexByte(text[index:], ';')
		if endOffset < 0 {
			result.WriteByte(text[index])
			index++
			continue
		}
		end := index + endOffset + 1
		encoded := text[index:end]
		if html.UnescapeString(encoded) == encoded {
			result.WriteString(encoded)
		} else {
			result.WriteString(strings.Repeat(" ", len([]rune(encoded))))
		}
		index = end
	}
	return result.String()
}
