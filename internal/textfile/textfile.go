// Package textfile decides which files in a vault are text the app can
// edit, and how their source is shown when they are not Markdown. Names
// come first, then a small read of the file's head for names that say
// nothing; the languages of the editor are the one table they name.
package textfile

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui/editor"
)

// SniffBytes is how much of a file's head is read before a name that says
// nothing is called text or binary.
const SniffBytes = 4096

// Reason is why a file is not opened in the editor.
type Reason int

const (
	// ReasonOK is a file the editor can take.
	ReasonOK Reason = iota
	// ReasonNotText is a file whose bytes are not text.
	ReasonNotText
	// ReasonTooLarge is a text file bigger than the editor will hold.
	ReasonTooLarge
	// ReasonEncoding is a file in an encoding the app will not guess.
	ReasonEncoding
)

// Info is what the app knows of a file after classifying it.
type Info struct {
	// Text is whether the file is read into the editor.
	Text bool
	// Language is the name a fenced code block and the editor agree on,
	// or "" for plain text.
	Language string
	// Encoding is how the bytes on disk become the text in the editor.
	Encoding Encoding
	// Reason says why a file that is not Text is left alone.
	Reason Reason
}

var binaryExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".webp": true, ".bmp": true, ".tiff": true, ".tif": true,
	".svg": true, ".avif": true, ".heic": true, ".ico": true,
	".pdf": true,
	".zip": true, ".tar": true, ".gz": true, ".bz2": true,
	".xz": true, ".7z": true, ".rar": true, ".tgz": true,
	".zst": true,
	".xls": true, ".xlsx": true, ".numbers": true,
	".mp3": true, ".wav": true, ".ogg": true, ".flac": true,
	".m4a": true, ".aac": true, ".opus": true,
	".mp4": true, ".mov": true, ".mkv": true, ".webm": true,
	".avi": true, ".m4v": true,
	".wasm": true, ".exe": true, ".dll": true, ".so": true,
	".dylib": true, ".bin": true, ".db": true, ".sqlite": true,
	".sqlite3": true, ".class": true, ".jar": true, ".war": true,
	".ear": true, ".o": true, ".a": true, ".pyc": true,
	".pyo": true, ".woff": true, ".woff2": true, ".ttf": true,
	".otf": true, ".eot": true,
}

var plainExts = map[string]bool{
	".txt": true, ".text": true, ".log": true, ".rtf": true,
	".csv": true, ".tsv": true, ".env": true, ".example": true,
	".gitignore": true, ".gitattributes": true, ".gitmodules": true,
	".dockerignore": true, ".npmignore": true, ".eslintignore": true,
	".prettierignore": true, ".nvmrc": true, ".python-version": true,
	".ruby-version": true, ".tool-versions": true, ".mailmap": true,
	".lock": true, ".map": true, ".out": true, ".jsonl": true,
	".ndjson": true,
}

// textNames are extensionless files that are text even when a small head
// of them would not make that plain. MyGo's own Filenames are asked first.
var textNames = map[string]bool{
	"license": true, "licence": true, "authors": true,
	"contributors": true, "notice": true, "todo": true,
	"procfile": true, "justfile": true, "vagrantfile": true,
	"brewfile": true, "cmakelists.txt": true,
}

// IsBinaryName reports whether the extension says the file is not text,
// whatever its first bytes happen to look like.
func IsBinaryName(path string) bool {
	return binaryExts[strings.ToLower(filepath.Ext(filepath.Base(path)))]
}

// ByName says what a file's name decides, and whether it decides at all.
// A name that is not known is left for Sniff to say.
func ByName(path string) (Info, bool) {
	base := filepath.Base(path)
	if IsBinaryName(base) {
		return Info{Reason: ReasonNotText}, true
	}
	if lang := editor.LanguageFor(base); lang != nil {
		return Info{Text: true, Language: fenceLanguage(lang.Name)}, true
	}
	ext := strings.ToLower(filepath.Ext(base))
	if plainExts[ext] || textNames[strings.ToLower(base)] {
		return Info{Text: true}, true
	}
	return Info{}, false
}

// Language is the name a fenced code block uses for path, or "" when the
// editor knows no language for it.
func Language(path string) string {
	info, _ := ByName(path)
	return info.Language
}

// fenceLanguage keeps only names editor.LanguageNamed will find again, so
// the preview asks for the same tokenizer as the editor.
func fenceLanguage(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if editor.LanguageNamed(name) != nil {
		return name
	}
	return ""
}

// Sniff says whether a file's head is text, and in which encoding. It is
// pure: the caller has already read the head.
func Sniff(head []byte) Info {
	if len(head) == 0 {
		return Info{Text: true, Encoding: UTF8}
	}
	switch {
	case bytes.HasPrefix(head, utf8BOM):
		return utf8Info(head[len(utf8BOM):], UTF8BOM)
	case bytes.HasPrefix(head, utf16LEBOM):
		return utf16Info(head[len(utf16LEBOM):], UTF16LE)
	case bytes.HasPrefix(head, utf16BEBOM):
		return utf16Info(head[len(utf16BEBOM):], UTF16BE)
	}
	if bytes.IndexByte(head, 0) >= 0 {
		return Info{Reason: ReasonNotText}
	}
	head = trimIncompleteUTF8(head)
	if !utf8.Valid(head) {
		return Info{Reason: ReasonEncoding}
	}
	if !mostlyText(string(head)) {
		return Info{Reason: ReasonNotText}
	}
	return Info{Text: true, Encoding: UTF8}
}

func utf8Info(head []byte, enc Encoding) Info {
	if bytes.IndexByte(head, 0) >= 0 {
		return Info{Reason: ReasonNotText}
	}
	head = trimIncompleteUTF8(head)
	if !utf8.Valid(head) {
		return Info{Reason: ReasonEncoding}
	}
	if !mostlyText(string(head)) {
		return Info{Reason: ReasonNotText}
	}
	return Info{Text: true, Encoding: enc}
}

func utf16Info(head []byte, enc Encoding) Info {
	if len(head)%2 != 0 {
		head = head[:len(head)-1]
	}
	text, err := decode16(head, enc == UTF16LE)
	if err != nil {
		return Info{Reason: ReasonEncoding}
	}
	if strings.ContainsRune(text, 0) || !mostlyText(text) {
		return Info{Reason: ReasonNotText}
	}
	return Info{Text: true, Encoding: enc}
}

// trimIncompleteUTF8 takes a trailing half-written rune off a head cut
// out of the middle of a file. Bytes wrong before that are left for utf8.
func trimIncompleteUTF8(head []byte) []byte {
	whole := head
	for i := 0; i < utf8.UTFMax-1 && len(head) > 0 && !utf8.Valid(head); i++ {
		r, size := utf8.DecodeLastRune(head)
		if r != utf8.RuneError || size != 1 {
			return whole
		}
		head = head[:len(head)-1]
	}
	if len(head) == 0 && len(whole) > 0 {
		return whole
	}
	return head
}

// mostlyText leaves tabs and line breaks alone, but a head made of other
// control characters is a binary file dressed as text.
func mostlyText(s string) bool {
	if s == "" {
		return true
	}
	bad, total := 0, 0
	for _, r := range s {
		total++
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' && r != '\f' {
			bad++
		} else if r == 0x7f {
			bad++
		}
	}
	return bad*100 <= total*30
}

// Classify reads no more than SniffBytes of path. maxBytes of zero leaves
// the size of the file unchecked.
func Classify(path string, maxBytes uint64) (Info, error) {
	info, known := ByName(path)
	if known && !info.Text {
		return info, nil
	}
	st, err := os.Stat(path)
	if err != nil {
		return Info{}, err
	}
	if st.IsDir() {
		return Info{Reason: ReasonNotText}, nil
	}
	if maxBytes > 0 && uint64(st.Size()) > maxBytes {
		return Info{Reason: ReasonTooLarge}, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return Info{}, err
	}
	defer f.Close()
	head := make([]byte, SniffBytes)
	n, err := f.Read(head)
	if err != nil && err != io.EOF {
		return Info{}, err
	}
	sniffed := Sniff(head[:n])
	if !sniffed.Text {
		return sniffed, nil
	}
	if !known {
		return sniffed, nil
	}
	info.Encoding = sniffed.Encoding
	return info, nil
}

// Fence writes text as a Markdown code block whose fence is longer than
// any run of backticks in the text itself.
func Fence(text, lang string) string {
	longest, run := 0, 0
	for _, r := range text {
		if r != '`' {
			run = 0
			continue
		}
		run++
		if run > longest {
			longest = run
		}
	}
	n := max(3, longest+1)
	fence := strings.Repeat("`", n)
	var b strings.Builder
	b.WriteString(fence)
	if lang != "" {
		b.WriteString(lang)
	}
	b.WriteByte('\n')
	b.WriteString(text)
	if !strings.HasSuffix(text, "\n") {
		b.WriteByte('\n')
	}
	b.WriteString(fence)
	b.WriteByte('\n')
	return b.String()
}
