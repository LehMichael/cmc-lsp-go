package textencoding

import (
	"bytes"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	tests := [][]byte{
		[]byte("plain UTF-8 ä"),
		append([]byte{0xEF, 0xBB, 0xBF}, []byte("BOM ä")...),
		{0x47, 0x72, 0xFC, 0xDF, 0x65},
	}
	for _, input := range tests {
		text, encoding := Decode(input)
		output, err := Encode(text, encoding)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(output, input) {
			t.Fatalf("round trip mismatch: %x != %x", output, input)
		}
	}
}

func TestDecodeWindows1252(t *testing.T) {
	input := []byte{'A', 0x93, 'x', 0x94, 0x96, 0x80}
	if got, want := DecodeWindows1252(input), "A“x”–€"; got != want {
		t.Fatalf("DecodeWindows1252() = %q, want %q", got, want)
	}
}
