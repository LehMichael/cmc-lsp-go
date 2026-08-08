// Package textencoding handles the two encodings supported by CMC.
package textencoding

import (
	"bytes"
	"errors"
	"unicode/utf8"
)

type Encoding int

const (
	UTF8 Encoding = iota
	UTF8BOM
	Latin1
)

func Decode(input []byte) (string, Encoding) {
	if bytes.HasPrefix(input, []byte{0xEF, 0xBB, 0xBF}) {
		return string(input[3:]), UTF8BOM
	}
	if utf8.Valid(input) {
		return string(input), UTF8
	}
	runes := make([]rune, len(input))
	for index, value := range input {
		runes[index] = rune(value)
	}
	return string(runes), Latin1
}

// DecodeWindows1252 decodes the encoding used by Siemens' parameter XML
// database. Bytes not assigned by Windows-1252 retain their control-code value.
func DecodeWindows1252(input []byte) string {
	runes := make([]rune, 0, len(input))
	for _, value := range input {
		if value >= 0x80 && value <= 0x9f {
			runes = append(runes, windows1252[value-0x80])
		} else {
			runes = append(runes, rune(value))
		}
	}
	return string(runes)
}

var windows1252 = [32]rune{
	'€', '\u0081', '‚', 'ƒ', '„', '…', '†', '‡',
	'ˆ', '‰', 'Š', '‹', 'Œ', '\u008d', 'Ž', '\u008f',
	'\u0090', '‘', '’', '“', '”', '•', '–', '—',
	'˜', '™', 'š', '›', 'œ', '\u009d', 'ž', 'Ÿ',
}

func Encode(input string, encoding Encoding) ([]byte, error) {
	switch encoding {
	case UTF8:
		return []byte(input), nil
	case UTF8BOM:
		return append([]byte{0xEF, 0xBB, 0xBF}, []byte(input)...), nil
	case Latin1:
		result := make([]byte, 0, len(input))
		for _, value := range input {
			if value > 0xFF {
				return nil, errors.New("formatted text contains a character that cannot be encoded as ISO-8859-1")
			}
			result = append(result, byte(value))
		}
		return result, nil
	default:
		return nil, errors.New("unknown text encoding")
	}
}
