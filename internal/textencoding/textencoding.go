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
