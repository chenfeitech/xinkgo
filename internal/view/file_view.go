package view

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/media"
	"github.com/helightxu/xinkgo/internal/textfile"
	"github.com/helightxu/xinkgo/internal/vault"
)

// What a file the app only shows is called in the window, and how much
// room its card keeps between its edges and its words.
const fileCardPad float32 = 24

// shownFile is a picture being loaded, or loaded.
type shownFile struct {
	mu   sync.Mutex
	src  ui.ImageSource
	err  error
	done bool
}

// shown is what the window has loaded of the files it shows: one entry
// to a path, kept until the file changes or is closed.
var shown sync.Map // path → *shownFile

// imageOf is the picture at path being made, or made: it is read and
// made on a goroutine of its own the first time, for a frame is no place
// to take a file off the disk.
func imageOf(c *ui.Context, path string) *shownFile {
	v, loaded := shown.LoadOrStore(path, &shownFile{})
	e := v.(*shownFile)
	if !loaded {
		services := c.Services()
		go func() {
			src, err := readImage(path)
			if err != nil {
				// Nothing to draw: a picture that failed to
				// be made is no picture, and not a nil one
				// dressed as one.
				src = nil
			}
			e.mu.Lock()
			e.src, e.err, e.done = src, err, true
			e.mu.Unlock()
			// The window is drawn again now that there is
			// something to draw.
			services.Invalidate()
		}()
	}
	return e
}

// readImage is the picture at path read off the disk and made into one
// the window can draw.
func readImage(path string) (ui.ImageSource, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(filepath.Ext(path), ".svg") {
		return ui.ParseSVG(data)
	}
	return ui.DecodeBitmap(data)
}

// forgetShown is what is let go of when a file has changed or been
// closed: what was made of it is made again.
func forgetShown(path string) {
	shown.Delete(path)
}

// filePane is what stands where the note does for a file the app cannot
// edit: the picture itself when it is one, and else a card saying what it
// is and what may be done with it.
func (w *Workspace) filePane(c *ui.Context, n *OpenNote) {
	if media.IsImage(n.Path) {
		w.imagePane(c, n)
		return
	}
	w.fileCard(c, n)
}

// imagePane is a picture as the note's column can hold it: as large as it
// is, and no larger than the room there is, as the preview shows it.
func (w *Workspace) imagePane(c *ui.Context, n *OpenNote) {
	th := c.Theme()
	ui.Column(c).Fill().Center().Padding(fileCardPad).Children(func() {
		e := imageOf(c, n.Path)
		e.mu.Lock()
		src, done := e.src, e.done
		e.mu.Unlock()
		switch {
		case src == nil && !done:
			ui.Text(c, locale.PictureComing()).TextColor(th.TextMuted)
			return
		case src == nil:
			// A picture the app was given but cannot make of
			// what it read: it is said so, and the machine's
			// own program is offered for it, rather than the
			// window waiting for a picture that never comes.
			ui.Text(c, locale.PictureNotShown()).
				TextColor(th.TextMuted)
			w.openOutsideButton(c, n.Path)
			return
		}
		img := ui.Image(c, src).Label(locale.A11yPicture(PathOf(n.Path))).
			Radius(th.Radius).MaxWidthPercent(100).MaxHeightPercent(100)
		var iw, ih float32
		switch s := src.(type) {
		case *ui.Bitmap:
			if s == nil {
				break
			}
			w, h := s.Size()
			iw, ih = float32(w), float32(h)
		case *ui.SVG:
			w, h := s.Size()
			iw, ih = w, h
		}
		if iw > 0 && ih > 0 {
			img.AspectRatio(iw / ih)
		}
	})
}

// fileCard is what the window says of a file it can neither edit nor
// show: what it is, why, and the way to have the machine open it with the
// program it has for such files.
func (w *Workspace) fileCard(c *ui.Context, n *OpenNote) {
	th := c.Theme()
	name := PathOf(n.Path)
	ui.Column(c).Fill().Center().Gap(8).Padding(fileCardPad).
		Children(func() {
			ui.Icon(c, iconOfMedia(n.Path)).Size(32, 32).
				TextColor(th.TextMuted)
			ui.Text(c, name).Bold()
			ui.Text(c, w.sayOfFile(n)).TextColor(th.TextMuted)
			w.openOutsideButton(c, n.Path)
		})
}

// openOutsideButton is the way to have the machine open the file with the
// program it has for it, where the window offers one.
func (w *Workspace) openOutsideButton(c *ui.Context, path string) {
	if w.OnOpenExternally == nil {
		return
	}
	if ui.PrimaryButton(c, "open-outside").
		Label(locale.OpenWithAnotherApp()).Clicked() {
		w.OnOpenExternally(path)
	}
}

// sayOfFile is what the window says of a file it cannot edit: a PDF is
// previewed by the machine's own program for them, and the rest are not
// shown at all.
func (w *Workspace) sayOfFile(n *OpenNote) string {
	switch {
	case media.IsPDF(n.Path):
		return locale.PdfPreviewed()
	case n.Info.Reason == textfile.ReasonTooLarge:
		return locale.FileTooLargeToEdit()
	case n.Info.Reason == textfile.ReasonEncoding:
		return locale.FileEncodingUnsupported()
	default:
		return locale.FileNotUsable()
	}
}

// fileBar is what the toolbar says of a file the app only shows: what it
// is, that it is not a thing to write in, and the way to open it with
// another program.
func (w *Workspace) fileBar(c *ui.Context, n *OpenNote) {
	ui.Row(c).Gap(8).Padding(4).AlignItems(ui.Center).Children(func() {
		ui.Text(c, media.KindOf(n.Path).String()).Bold()
		ui.Text(c, locale.ShownNotEdited()).TextColor(
			c.Theme().TextMuted).Grow(1)
		if w.OnOpenExternally != nil {
			if ui.Button(c.Key("file-outside"),
				locale.OpenWithAnotherApp()).
				Clicked() {
				w.OnOpenExternally(n.Path)
			}
		}
	})
}

// iconOfMedia is the icon a file the app only shows is drawn by: the one
// the tree gives it, for the two are of one mind about what a file is.
func iconOfMedia(path string) *ui.SVG {
	e := &vault.Entry{Name: filepath.Base(path), Path: path}
	return iconOf(e, false)
}
