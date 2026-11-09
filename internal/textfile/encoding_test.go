package textfile

import (
	"bytes"
	"testing"
)

func TestEncodeDecode_keepsTheMarkAndTheLineEnds(t *testing.T) {
	text := "hello\r\n\u4e16\u754c\n"
	for _, c := range []struct {
		enc  Encoding
		mark []byte
	}{
		{UTF8, nil},
		{UTF8BOM, utf8BOM},
		{UTF16LE, utf16LEBOM},
		{UTF16BE, utf16BEBOM},
	} {
		data, err := Encode(text, c.enc)
		if err != nil {
			t.Fatalf("Encode(%v): %v", c.enc, err)
		}
		if len(c.mark) > 0 && !bytes.HasPrefix(data, c.mark) {
			t.Errorf("Encode(%v) lost its mark", c.enc)
		}
		if got := Detect(data); got != c.enc {
			t.Errorf("Detect(Encode(%v)) = %v, want %v", c.enc, got,
				c.enc)
		}
		again, err := Decode(data, c.enc)
		if err != nil {
			t.Fatalf("Decode(%v): %v", c.enc, err)
		}
		if again != text {
			t.Errorf("Decode(Encode(%v)) = %q, want %q", c.enc, again,
				text)
		}
	}
}

func TestDecode_anOddUTF16FileIsNotReadAsText(t *testing.T) {
	if _, err := Decode([]byte{0xFF, 0xFE, 'a'}, UTF16LE); err == nil {
		t.Error("an odd number of UTF-16 bytes was decoded")
	}
}
