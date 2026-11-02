package view

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
)

// fileOf writes a file of that name in the folder, and gives its path.
func fileOf(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// littlePicture is a picture of two pixels the app can decode.
func littlePicture(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.Set(0, 0, color.RGBA{R: 200, G: 100, B: 50, A: 255})
	var buf []byte
	w := &pngWriter{&buf}
	if err := png.Encode(w, img); err != nil {
		t.Fatal(err)
	}
	return buf
}

// pngWriter is a writer png.Encode will take.
type pngWriter struct{ buf *[]byte }

func (w *pngWriter) Write(p []byte) (int, error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}

// windowOfFile is a window with one file open in it, and the preview
// beside it.
func windowOfFile(t *testing.T, name string, data []byte) (
	*Workspace, *OpenNote, *ui.Tester,
) {
	t.Helper()
	dir := t.TempDir()
	path := fileOf(t, dir, name, data)
	w := NewWorkspace(config.Default())
	n, err := w.Tabs.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	w.Layout.Left = 0
	w.Layout.Right = 400
	return w, n, ui.NewTester(w.View, 1000, 600)
}

// TestOpenFile_aPictureIsShownAndNotEdited checks the way a picture is
// opened: it stands open as a file of its own, but nothing of it is read
// into an editor, and it is never written back.
func TestOpenFile_aPictureIsShownAndNotEdited(t *testing.T) {
	_, n, _ := windowOfFile(t, "pic.png", littlePicture(t))
	if !n.Preview {
		t.Error("the picture was opened as a note to edit")
	}
	if n.Note.Text != "" {
		t.Errorf("the picture's %d bytes were read into the editor",
			len(n.Note.Text))
	}
	if n.Note.Dirty {
		t.Error("the picture was opened with changes to write")
	}
}

// TestOpenFile_aPdfIsShownAndNotEdited checks the same of a PDF, which is
// the file most likely to be a hundred megabytes of bytes no one can
// edit.
func TestOpenFile_aPdfIsShownAndNotEdited(t *testing.T) {
	_, n, _ := windowOfFile(t, "paper.pdf", []byte("%PDF-1.7\n%\xe2\xe3\n"))
	if !n.Preview {
		t.Error("the PDF was opened as a note to edit")
	}
	if n.Note.Text != "" {
		t.Error("the PDF was read into the editor")
	}
}

// TestFilePane_aPictureIsDrawn checks what stands where the note does:
// the picture itself, and not the editor.
func TestFilePane_aPictureIsDrawn(t *testing.T) {
	w, n, te := windowOfFile(t, "pic.png", littlePicture(t))
	// The picture is read and made on another goroutine: give it the
	// moments it takes, as a user's eye would.
	for i := 0; i < 30; i++ {
		te.Advance(20 * time.Millisecond)
		te.Frame()
		if _, ok := te.Find("picture pic.png"); ok {
			break
		}
	}
	if _, ok := te.Find("picture pic.png"); !ok {
		t.Errorf("the window shows %q, want the picture itself",
			te.Texts())
	}
	if te.HasText("the picture is coming") {
		t.Error("the picture never came")
	}
	if te.HasText("No note open") {
		t.Error("the window shows an empty page instead of the picture")
	}
	if w.Tabs.ActiveNote() != n {
		t.Error("the picture is not the file that is open")
	}
}

// TestFilePane_aFileThatCannotBeShownSaysSo checks the rest: a file the
// app can neither edit nor show is said to be so, and the window gives
// the way to have the machine open it.
func TestFilePane_aFileThatCannotBeShownSaysSo(t *testing.T) {
	asked := ""
	w, _, te := windowOfFile(t, "thing.zip", []byte("PK\x03\x04"))
	w.OnOpenExternally = func(path string) { asked = path }
	te.Frame()
	if !te.HasText("this file can neither be edited nor shown here") {
		t.Errorf("the window says %q, want it said that it cannot",
			te.Texts())
	}
	if err := te.Click("Open with another app"); err != nil {
		t.Fatalf("there is no way to open the file with another app: %v",
			err)
	}
	if asked == "" {
		t.Error("the machine was not asked to open the file")
	}
}

// TestFilePane_aPdfIsPreviewedByTheMachine checks what a PDF is: it is
// not drawn here, and the way to see it is the program the machine has
// for it.
func TestFilePane_aPdfIsPreviewedByTheMachine(t *testing.T) {
	asked := ""
	w, _, te := windowOfFile(t, "paper.pdf", []byte("%PDF-1.7\n"))
	w.OnOpenExternally = func(path string) { asked = path }
	te.Frame()
	if !te.HasText("a PDF is previewed by the program the machine has " +
		"for it") {
		t.Errorf("the window says %q, want it said how a PDF is seen",
			te.Texts())
	}
	if err := te.Click("Open with another app"); err != nil {
		t.Fatalf("there is no way to open the PDF: %v", err)
	}
	if asked == "" {
		t.Error("the machine was not asked to open the PDF")
	}
}

// TestPreviewTextOfFile_thePictureItself checks what the preview is
// given: the picture, and not the bytes of it read as Markdown.
func TestPreviewTextOfFile_thePictureItself(t *testing.T) {
	w, n, _ := windowOfFile(t, "pic.png", littlePicture(t))
	got := w.previewText(n)
	if got != "![<pic.png>](<pic.png>)" {
		t.Errorf("preview = %q, want the picture itself", got)
	}
}

// TestOpenFile_aNoteIsStillEdited checks the other side of the line: a
// note is read and written as it always was, and is not one of the files
// the app only shows.
func TestOpenFile_aNoteIsStillEdited(t *testing.T) {
	_, n, _ := windowOfFile(t, "a.md", []byte("# a note\n"))
	if n.Preview {
		t.Error("the note was opened as a file to show only")
	}
	if n.Note.Text != "# a note\n" {
		t.Errorf("text = %q, want the note read as it stands", n.Note.Text)
	}
}

// TestFilePane_aPictureThatCannotBeMade checks a file that is a picture
// by its name but that the app cannot make one of, as a HEIC is: the
// window says so and offers the machine's own program, rather than
// waiting for a picture that never comes.
func TestFilePane_aPictureThatCannotBeMade(t *testing.T) {
	asked := ""
	w, _, te := windowOfFile(t, "pic.heic", []byte("not a picture"))
	w.OnOpenExternally = func(path string) { asked = path }
	for i := 0; i < 30; i++ {
		te.Advance(20 * time.Millisecond)
		te.Frame()
		if te.HasText("this picture cannot be shown here") {
			break
		}
	}
	if !te.HasText("this picture cannot be shown here") {
		t.Errorf("the window says %q, want it said that it cannot",
			te.Texts())
	}
	if err := te.Click("Open with another app"); err != nil {
		t.Fatalf("there is no way to open the picture: %v", err)
	}
	if asked == "" {
		t.Error("the machine was not asked to open the picture")
	}
}
