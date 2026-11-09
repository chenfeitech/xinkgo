package view

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/textfile"
)

// textWindow is a window with one of the vault's files open, and a limit
// on how large a text file the editor will hold.
func textWindow(t *testing.T, name string, data []byte, max uint64) (
	*Workspace, *OpenNote, *ui.Tester,
) {
	t.Helper()
	path := fileOf(t, t.TempDir(), name, data)
	w := NewWorkspace(config.Default())
	w.Tabs.MaxTextFileBytes = max
	n, err := w.Tabs.Open(path)
	if err != nil {
		t.Fatalf("Open %s: %v", name, err)
	}
	w.Layout.Left = 0
	w.Layout.Right = 400
	return w, n, ui.NewTester(w.View, 1000, 600)
}

func editorLanguage(n *OpenNote) string {
	if lang := n.Editor.Buffer.Language(); lang != nil {
		return strings.ToLower(lang.Name)
	}
	return ""
}

// TestTabs_openTextFilesWithTheirLanguages checks the editor's one way
// in: a text file is read, given the language MyGo knows for it, and not
// left as a picture or a card.
func TestTabs_openTextFilesWithTheirLanguages(t *testing.T) {
	cases := []struct {
		name, data, language string
	}{
		{"index.html", "<h1>hi</h1>\n", "html"},
		{"app.js", "const x = 1;\n", "javascript"},
		{"app.mjs", "export const x = 1;\n", "javascript"},
		{"theme.css", "body { color: red; }\n", "css"},
		{"theme.scss", "$c: red;\n", "css"},
		{"App.vue", "<template></template>\n", "html"},
		{"data.json", "{}\n", "json"},
		{"Makefile", "all:\n\techo hi\n", "makefile"},
		{"Dockerfile", "FROM scratch\n", "dockerfile"},
		{"LICENSE", "MIT\n", ""},
		{".env", "A=1\n", ""},
		{".gitignore", "*.log\n", ""},
		{"notes.txt", "hello\n", ""},
		{"table.csv", "a,b\n", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := NewWorkspace(config.Default())
			path := fileOf(t, t.TempDir(), c.name, []byte(c.data))
			n, err := w.Tabs.Open(path)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			if n.Preview || !n.Info.Text {
				t.Fatalf("%q was not opened as text: %+v", c.name, n.Info)
			}
			if n.Note.Text != c.data {
				t.Errorf("text = %q, want %q", n.Note.Text, c.data)
			}
			if got := editorLanguage(n); got != c.language {
				t.Errorf("language = %q, want %q", got, c.language)
			}
			if n.Info.Language != c.language {
				t.Errorf("info language = %q, want %q", n.Info.Language,
					c.language)
			}
		})
	}
}

// TestTabs_aFileTheEditorCannotHoldSaysWhy checks the three reasons a
// file is shown rather than edited, and that none of them is read into
// an editor.
func TestTabs_aFileTheEditorCannotHoldSaysWhy(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		max  uint64
		why  textfile.Reason
	}{
		{"blob", []byte{'a', 0, 'b'}, config.DefaultMaxTextFileBytes,
			textfile.ReasonNotText},
		{"large.txt", []byte("0123456789"), 8, textfile.ReasonTooLarge},
		{"bad.txt", []byte{0x81, 0x82, 0x83},
			config.DefaultMaxTextFileBytes, textfile.ReasonEncoding},
	}
	for _, c := range cases {
		_, n, _ := textWindow(t, c.name, c.data, c.max)
		if !n.Preview {
			t.Errorf("%q was opened in the editor", c.name)
		}
		if n.Info.Reason != c.why {
			t.Errorf("%q says %v, want %v", c.name, n.Info.Reason, c.why)
		}
		if n.Note.Text != "" {
			t.Errorf("%q holds %q of text", c.name, n.Note.Text)
		}
	}
}

// TestFileCard_theReasonIsSaid checks the words a card gives a file that
// is too large, in an encoding the app will not guess, or not text at all.
func TestFileCard_theReasonIsSaid(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		max  uint64
		want string
	}{
		{"large.txt", []byte("0123456789"), 8,
			locale.FileTooLargeToEdit()},
		{"bad.txt", []byte{0x81, 0x82, 0x83},
			config.DefaultMaxTextFileBytes,
			locale.FileEncodingUnsupported()},
		{"blob", []byte{'a', 0, 'b'}, config.DefaultMaxTextFileBytes,
			locale.FileNotUsable()},
	}
	for _, c := range cases {
		_, _, te := textWindow(t, c.name, c.data, c.max)
		te.Frame()
		if !te.HasText(c.want) {
			t.Errorf("%q says %q, want %q", c.name, te.Texts(), c.want)
		}
	}
}

// TestPreviewText_sourceIsFencedAndNotLinked checks that a code file is
// shown as source, with the language of its name, and not written out as
// Markdown with its links rewritten.
func TestPreviewText_sourceIsFencedAndNotLinked(t *testing.T) {
	cases := []struct {
		name, data, fence string
	}{
		{"index.html", "<h1>hi</h1>\n", "```html\n"},
		{"app.js", "const x = 1;\n", "```javascript\n"},
		{"theme.css", "body {}\n", "```css\n"},
		{"Makefile", "all:\n", "```makefile\n"},
		{"LICENSE", "MIT\n", "```\n"},
		{"data.json", "{}\n", "```json\n"},
	}
	for _, c := range cases {
		w, n, _ := textWindow(t, c.name, []byte(c.data),
			config.DefaultMaxTextFileBytes)
		got := w.previewText(n)
		if !strings.HasPrefix(got, c.fence) {
			t.Errorf("%q preview opens %q, want %q", c.name, got,
				c.fence)
		}
		if !strings.Contains(got, c.data) {
			t.Errorf("%q preview changed the source: %q", c.name, got)
		}
	}

	// A link in double brackets is a link only in a note. With the index
	// knowing b, the old Markdown path would have written xinkgo:b here.
	w, _, _ := windowWithTwoNotes(t)
	path := fileOf(t, t.TempDir(), "app.js",
		[]byte("const b = [[b]];\n"))
	n, err := w.Tabs.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got := w.previewText(n)
	if !strings.Contains(got, "[[b]]") || strings.Contains(got, "xinkgo:") {
		t.Errorf("source was rewritten as a note: %q", got)
	}
}

// TestPreviewText_backticksInSourceDoNotCloseTheFence checks a fence
// longer than any run of backticks in the source itself.
func TestPreviewText_backticksInSourceDoNotCloseTheFence(t *testing.T) {
	source := "const s = `x`;\n```\nnot a fence\n"
	w, n, _ := textWindow(t, "app.js", []byte(source),
		config.DefaultMaxTextFileBytes)
	got := w.previewText(n)
	if !strings.HasPrefix(got, "````javascript\n") {
		t.Errorf("preview opens with %q, want four backticks", got)
	}
	if !strings.Contains(got, "not a fence\n") {
		t.Errorf("the source was cut short: %q", got)
	}
}

// TestPreviewText_aNoteIsStillMarkdown checks the one kind the preview
// still renders rather than showing as source.
func TestPreviewText_aNoteIsStillMarkdown(t *testing.T) {
	w, n, _ := textWindow(t, "a.md", []byte("# a note\n"),
		config.DefaultMaxTextFileBytes)
	if got := w.previewText(n); got != "# a note\n" {
		t.Errorf("note preview = %q, want it as it stands", got)
	}
}

// TestExportText_aCodeFileIsExportedAsThePreviewReadsIt checks the
// promise ExportText makes: what the preview reads is what is written out.
func TestExportText_aCodeFileIsExportedAsThePreviewReadsIt(t *testing.T) {
	w, _, _ := textWindow(t, "app.js", []byte("const x = 1;\n"),
		config.DefaultMaxTextFileBytes)
	path, text, ok := w.ExportText()
	if !ok {
		t.Fatal("a code file has nothing to write out")
	}
	if filepath.Ext(path) != ".js" {
		t.Errorf("path = %q, want the code file", path)
	}
	if !strings.HasPrefix(text, "```javascript\n") {
		t.Errorf("export = %q, want the fenced source", text)
	}
}

// TestToolbar_aTextFileHasNoMarkdownButtons checks the toolbar says what
// a source file is, rather than offering marks that write Markdown.
func TestToolbar_aTextFileHasNoMarkdownButtons(t *testing.T) {
	w, _, te := textWindow(t, "app.js", []byte("const x = 1;\n"),
		config.DefaultMaxTextFileBytes)
	w.Layout.Right = 0
	te = ui.NewTester(w.View, 800, 400)
	te.Frame()
	if !te.HasText(locale.TextFileShownAsSource()) {
		t.Errorf("the toolbar says %q, want the source note", te.Texts())
	}
	if _, ok := te.Find(locale.ToolbarMarkdown()); ok {
		t.Error("a source file was given the Markdown toolbar")
	}
}
