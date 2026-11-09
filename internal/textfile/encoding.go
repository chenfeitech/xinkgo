package textfile

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// Encoding is how the bytes of a text file become the string the editor
// holds, and become bytes again when it is saved.
type Encoding int

// EncodingAuto asks Open to read the mark from the bytes of the file.
const EncodingAuto Encoding = -1

const (
	// UTF8 is text without a byte order mark.
	UTF8 Encoding = iota
	// UTF8BOM is UTF-8 with the three-byte mark kept on disk.
	UTF8BOM
	// UTF16LE is UTF-16 little endian with its mark.
	UTF16LE
	// UTF16BE is UTF-16 big endian with its mark.
	UTF16BE
)

var (
	utf8BOM    = []byte{0xEF, 0xBB, 0xBF}
	utf16LEBOM = []byte{0xFF, 0xFE}
	utf16BEBOM = []byte{0xFE, 0xFF}
	errUTF16   = errors.New("invalid UTF-16")
)

// Detect says the encoding a file's byte order mark names, or UTF-8 when
// it has none.
func Detect(data []byte) Encoding {
	switch {
	case bytes.HasPrefix(data, utf16LEBOM):
		return UTF16LE
	case bytes.HasPrefix(data, utf16BEBOM):
		return UTF16BE
	case bytes.HasPrefix(data, utf8BOM):
		return UTF8BOM
	default:
		return UTF8
	}
}

// Decode turns the bytes of a file into the text the editor holds.
func Decode(data []byte, enc Encoding) (string, error) {
	switch enc {
	case UTF8BOM:
		data = bytes.TrimPrefix(data, utf8BOM)
	case UTF16LE:
		data = bytes.TrimPrefix(data, utf16LEBOM)
		return decode16(data, true)
	case UTF16BE:
		data = bytes.TrimPrefix(data, utf16BEBOM)
		return decode16(data, false)
	}
	return string(data), nil
}

// Encode turns the editor's text back into the bytes of its file, keeping
// the byte order mark the file was opened with.
func Encode(text string, enc Encoding) ([]byte, error) {
	switch enc {
	case UTF8BOM:
		return append(append([]byte{}, utf8BOM...), []byte(text)...), nil
	case UTF16LE, UTF16BE:
		units := utf16.Encode([]rune(text))
		mark := utf16LEBOM
		if enc == UTF16BE {
			mark = utf16BEBOM
		}
		out := make([]byte, len(mark)+2*len(units))
		copy(out, mark)
		for i, u := range units {
			if enc == UTF16LE {
				binary.LittleEndian.PutUint16(out[len(mark)+2*i:], u)
			} else {
				binary.BigEndian.PutUint16(out[len(mark)+2*i:], u)
			}
		}
		return out, nil
	default:
		return []byte(text), nil
	}
}

func decode16(data []byte, little bool) (string, error) {
	if len(data)%2 != 0 {
		return "", errUTF16
	}
	units := make([]uint16, len(data)/2)
	for i := range units {
		if little {
			units[i] = binary.LittleEndian.Uint16(data[2*i:])
		} else {
			units[i] = binary.BigEndian.Uint16(data[2*i:])
		}
	}
	text := string(utf16.Decode(units))
	if strings.ContainsRune(text, utf8.RuneError) {
		return "", errUTF16
	}
	return text, nil
}
