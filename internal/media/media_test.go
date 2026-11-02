package media

import (
	"path/filepath"
	"testing"
)

// TestKindOf checks what the tree calls each kind of file.
func TestKindOf(t *testing.T) {
	cases := map[string]Kind{
		"a.md":        KindMarkdown,
		"a.markdown":  KindMarkdown,
		"a.png":       KindImage,
		"a.JPG":       KindImage, // the case of the extension says nothing
		"main.go":     KindCode,
		"a.json":      KindData,
		"a.yaml":      KindData,
		"a.toml":      KindData,
		"a.zip":       KindArchive,
		"a.csv":       KindSpreadsheet,
		"a.mp3":       KindAudio,
		"a.mp4":       KindVideo,
		"a.unknown":   KindOther,
		"Makefile":    KindOther,
		"notes/.xink": KindDir,
	}
	for path, want := range cases {
		if got := KindOf(path); got != want {
			t.Errorf("KindOf(%q) = %v, want %v", path, got, want)
		}
	}
}

// TestIsEditableText checks what the app opens in the editor.
func TestIsEditableText(t *testing.T) {
	editable := []string{"a.md", "a.txt", "a.json", "a.yml", "a.toml",
		"main.go", "a.rs"}
	for _, path := range editable {
		if !IsEditableText(path) {
			t.Errorf("%q should be editable", path)
		}
	}
	notEditable := []string{"a.png", "a.zip", "a.mp4", "a.unknown"}
	for _, path := range notEditable {
		if IsEditableText(path) {
			t.Errorf("%q should not be editable", path)
		}
	}
}

// TestIsImage checks the pictures the preview shows.
func TestIsImage(t *testing.T) {
	for _, path := range []string{"a.png", "a.jpeg", "a.webp", "a.svg",
		"a.PNG"} {
		if !IsImage(path) {
			t.Errorf("%q should be an image", path)
		}
	}
	for _, path := range []string{"a.md", "a.pdf", "a.txt"} {
		if IsImage(path) {
			t.Errorf("%q should not be an image", path)
		}
	}
}

// TestLanguage checks the language the editor highlights a file as.
func TestLanguage(t *testing.T) {
	cases := map[string]string{
		"a.md":       "markdown",
		"a.markdown": "markdown",
		"a.json":     "json",
		"a.yaml":     "yaml",
		"a.yml":      "yaml",
		"a.toml":     "toml",
		"main.go":    "go",
		"a.rs":       "rust",
		"a.txt":      "",
		"a.unknown":  "",
	}
	for path, want := range cases {
		if got := Language(path); got != want {
			t.Errorf("Language(%q) = %q, want %q", path, got, want)
		}
	}
}

// TestIsAIDir checks that the AI's own directory is left alone wherever
// it appears.
func TestIsAIDir(t *testing.T) {
	for _, path := range []string{".xink", "vault/.xink",
		filepath.Join("vault", ".xink", "notes")} {
		if !IsAIDir(path) {
			t.Errorf("%q should be the AI's directory", path)
		}
	}
	for _, path := range []string{"xink", "vault/notes", ".xink.md"} {
		if IsAIDir(path) {
			t.Errorf("%q is not the AI's directory", path)
		}
	}
}

// TestKind_string checks the name a file is headed with in the file
// view, which is what the kind is called in the language the app shows.
func TestKind_string(t *testing.T) {
	if KindMarkdown.String() != "Markdown" || KindDir.String() != "Folder" {
		t.Errorf("kinds are named %q and %q", KindMarkdown, KindDir)
	}
}

// TestKindOf_aPdfIsAKindOfItsOwn checks the file the app has no renderer
// for, and so shows as the machine does: it is told from the rest, for
// what is done with it is not what is done with any of them.
func TestKindOf_aPdfIsAKindOfItsOwn(t *testing.T) {
	if got := KindOf("/vault/paper.pdf"); got != KindPDF {
		t.Errorf("kind = %v, want pdf", got)
	}
	if !IsPDF("/vault/paper.PDF") {
		t.Error("a PDF written in capitals was not known for one")
	}
	if IsPDF("/vault/paper.png") {
		t.Error("a picture was taken for a PDF")
	}
}

// TestIsEditableText_whatTheEditorIsFor checks the line the window draws
// between what it edits and what it only shows: a note, a text and the
// files of data, and none of the pictures, the PDFs or the archives.
func TestIsEditableText_whatTheEditorIsFor(t *testing.T) {
	editable := []string{"a.md", "a.markdown", "a.txt", "a.log",
		"a.json", "a.yaml", "a.toml", "a.go", "a.py"}
	for _, name := range editable {
		if !IsEditableText(name) {
			t.Errorf("%v is not opened in the editor", name)
		}
	}
	shown := []string{"a.png", "a.jpg", "a.svg", "a.pdf", "a.zip",
		"a.mp4", "a.mp3", "a.xlsx"}
	for _, name := range shown {
		if IsEditableText(name) {
			t.Errorf("%v was opened in the editor", name)
		}
	}
}
