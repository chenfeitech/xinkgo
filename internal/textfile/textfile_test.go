package textfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestByName_theEditorsLanguagesAreTheOneTable(t *testing.T) {
	cases := []struct {
		path, language string
		text           bool
	}{
		{"index.html", "html", true},
		{"app.js", "javascript", true},
		{"app.mjs", "javascript", true},
		{"theme.css", "css", true},
		{"theme.scss", "css", true},
		{"App.vue", "html", true},
		{"data.json", "json", true},
		{"a.md", "markdown", true},
		{"Makefile", "makefile", true},
		{"Dockerfile", "dockerfile", true},
		{"notes.txt", "", true},
		{"table.csv", "", true},
		{".env", "", true},
		{"LICENSE", "", true},
		{"pic.svg", "", false},
		{"a.zip", "", false},
		{"paper.pdf", "", false},
		{"movie.mp4", "", false},
	}
	for _, c := range cases {
		info, known := ByName(c.path)
		if !known {
			t.Errorf("ByName(%q) says the name is unknown", c.path)
			continue
		}
		if info.Text != c.text || info.Language != c.language {
			t.Errorf("ByName(%q) = %+v, want text=%v language=%q",
				c.path, info, c.text, c.language)
		}
	}
	if _, known := ByName("mystery"); known {
		t.Error("an unknown name was decided without reading the file")
	}
}

func TestSniff_textBinaryAndEncoding(t *testing.T) {
	cases := []struct {
		name string
		head []byte
		text bool
		enc  Encoding
		why  Reason
	}{
		{"empty", nil, true, UTF8, ReasonOK},
		{"ascii", []byte("all:\n\techo hi\n"), true, UTF8, ReasonOK},
		{"utf8 bom", append(append([]byte{}, utf8BOM...), []byte("hi")...),
			true, UTF8BOM, ReasonOK},
		{"utf16le", mustEncode(t, "hi\r\n", UTF16LE), true, UTF16LE, ReasonOK},
		{"utf16be", mustEncode(t, "hi\r\n", UTF16BE), true, UTF16BE, ReasonOK},
		{"nul", []byte{'a', 0, 'b'}, false, UTF8, ReasonNotText},
		{"bad utf8", []byte{0x81, 0x82, 0x83}, false, UTF8, ReasonEncoding},
		{"controls", []byte{1, 2, 3, 4, 'a'}, false, UTF8, ReasonNotText},
	}
	for _, c := range cases {
		got := Sniff(c.head)
		if got.Text != c.text || got.Encoding != c.enc || got.Reason != c.why {
			t.Errorf("Sniff(%s) = %+v, want text=%v enc=%v why=%v",
				c.name, got, c.text, c.enc, c.why)
		}
	}
}

func TestClassify_readsOnlyTheHead(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, data []byte, max uint64) Info {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		info, err := Classify(path, max)
		if err != nil {
			t.Fatalf("Classify(%s): %v", name, err)
		}
		return info
	}
	if got := write("Makefile", []byte("all:\n"), 0); !got.Text ||
		got.Language != "makefile" || got.Encoding != UTF8 {
		t.Errorf("Makefile = %+v, want makefile text", got)
	}
	if got := write("LICENSE", []byte("MIT\n"), 0); !got.Text ||
		got.Language != "" {
		t.Errorf("LICENSE = %+v, want plain text", got)
	}
	if got := write("blob", []byte{'a', 0, 'b'}, 0); got.Text ||
		got.Reason != ReasonNotText {
		t.Errorf("blob = %+v, want not text", got)
	}
	if got := write("bom.txt", append(append([]byte{}, utf8BOM...),
		[]byte("hi")...), 0); !got.Text || got.Encoding != UTF8BOM {
		t.Errorf("bom.txt = %+v, want UTF-8 BOM", got)
	}
	if got := write("large.txt", []byte("0123456789"), 8); got.Text ||
		got.Reason != ReasonTooLarge {
		t.Errorf("large.txt = %+v, want too large", got)
	}
	if got := write("wide.txt", mustEncode(t, "hi", UTF16LE), 0); !got.Text || got.Encoding != UTF16LE {
		t.Errorf("wide.txt = %+v, want UTF-16LE", got)
	}
}

func TestFence_aRunOfBackticksDoesNotCloseIt(t *testing.T) {
	text := "const s = `x`;\n```\nnot a fence\n````\nend"
	got := Fence(text, "javascript")
	if !strings.HasPrefix(got, "`````javascript\n") {
		t.Errorf("fence opens with %q, want five backticks", got)
	}
	if !strings.HasSuffix(got, "`````\n") {
		t.Errorf("fence closes with %q, want five backticks", got)
	}
	if !strings.Contains(got, text) {
		t.Error("the source was changed inside the fence")
	}
	if plain := Fence("one line", ""); plain != "```\none line\n```\n" {
		t.Errorf("plain fence = %q", plain)
	}
}

func mustEncode(t *testing.T, text string, enc Encoding) []byte {
	t.Helper()
	data, err := Encode(text, enc)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
