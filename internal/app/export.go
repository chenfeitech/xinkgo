package app

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/egoist/mygo"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/export"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/view"
)

// exportNote writes the note that is open out as a file the user names: a
// page of HTML that stands on its own, or the PDF the machine's own
// webview prints of that very page.
//
// What is written out is what the preview reads, and nothing else, so
// that it is what the user sees: the words of it are the parser's, which
// draws the preview too, and the pictures are read from the note's own
// folder, as they are for the preview.
func (s *appState) exportNote(kind string) {
	path, text, ok := s.ws.ExportText()
	if !ok {
		s.say(locale.SayExportNoNote())
		return
	}
	// A PDF is printed by the webview of the machine: where there is
	// none, there is nothing to print with, and the page is all there is.
	if kind == view.ExportPDF && !mygo.WebViewAvailable() {
		s.say(locale.SayExportNoWebView())
		return
	}
	go func() {
		ext := "html"
		if kind == view.ExportPDF {
			ext = "pdf"
		}
		stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		dest, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Title:       exportTitle(kind),
			DefaultPath: filepath.Join(filepath.Dir(path), stem+"."+ext),
			Filters: []mygo.FileFilter{
				{Name: strings.ToUpper(ext), Extensions: []string{ext}},
			},
		})
		if err != nil || dest == "" {
			return
		}
		page, err := s.pageOf(text, path, filepath.Dir(dest),
			kind == view.ExportPDF)
		if err != nil {
			s.exportFailed("the note could not be laid out", err)
			return
		}
		if kind == view.ExportPDF {
			s.printPDF(page, dest)
			return
		}
		if err := os.WriteFile(dest, page, 0o644); err != nil {
			s.exportFailed("the page could not be written", err)
			return
		}
		s.say(locale.SayExportWritten(filepath.Base(dest)))
	}()
}

// exportTitle is what the dialog asks: where the page is to be put, or
// the PDF of it.
func exportTitle(kind string) string {
	if kind == view.ExportPDF {
		return locale.DialogSavePdf()
	}
	return locale.DialogSavePage()
}

// pageOf is the note as a page of HTML: in the colors of the scheme the
// window is drawn in, and in its light colors when it is bound for paper,
// for a dark page prints as a dark sheet, and the words of a dark scheme
// are light ones, which on white paper say nothing at all.
func (s *appState) pageOf(text, path, dir string, paper bool) ([]byte, error) {
	cfg := s.config
	images := export.ImagesEmbed
	switch cfg.Export.Images {
	case config.ExportImagesCopy:
		images = export.ImagesCopy
	case config.ExportImagesNone:
		images = export.ImagesNone
	}
	o := export.Options{
		Title: strings.TrimSuffix(filepath.Base(path),
			filepath.Ext(path)),
		Scheme:  cfg.Basic.Theme,
		Dark:    !paper && mygo.Theme.IsDark(),
		Font:    cssFont(cfg.Preview.Font.Family),
		BaseDir: filepath.Dir(path),
		Images:  images,
	}
	if cfg.Basic.Language == config.LanguageChinese {
		o.Lang = "zh-CN"
	}
	if images == export.ImagesCopy {
		o.Copy = func(name string, data []byte) error {
			to := filepath.Join(dir, name)
			if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
				return err
			}
			return os.WriteFile(to, data, 0o644)
		}
	}
	return export.HTML(text, o)
}

// printPDF is a page printed onto paper: the webview of the machine lays
// it out, breaks it into pages, and gives back the PDF — and it is the
// webview that has the fonts for Chinese, and the rules that keep a table
// from being cut in two. It prints from a window that is never shown.
//
// The page it prints is written beside the PDF it becomes, and it is taken
// away when the printing is over, and not when this function returns: the
// window reads the file in its own time, and a page that is gone by then
// is a page that is never printed at all. Beside the PDF it is written,
// and not into the machine's own folder of temporary files, because a note
// whose pictures stand beside the page names them by a path relative to
// it, and a webview reads them only from where the page is.
func (s *appState) printPDF(page []byte, dest string) {
	name, err := writePage(filepath.Dir(dest), page)
	if err != nil {
		s.exportFailed("the page could not be printed", err)
		return
	}
	size := mygo.PageA4
	if s.config.Export.PageSize == config.ExportPageLetter {
		size = mygo.PageLetter
	}
	// A window is what prints, and it is made on the main thread, as
	// every window is; the printing itself is asked from wherever it
	// likes, and comes back on its own.
	mygo.RunOnMain(func() {
		win := mygo.NewWindow(mygo.WindowOptions{
			Hidden: true, Width: 794, Height: 1123,
		})
		var once sync.Once
		// The page is taken away and the window closed once, whichever
		// of the two ends the printing comes by.
		over := func() {
			once.Do(func() {
				os.Remove(name)
				mygo.RunOnMain(func() { win.Close() })
			})
		}
		var offShown, offFailed func()
		offShown = win.OnReadyToShow(func() {
			offShown()
			go func() {
				pdf, err := win.Page().PrintToPDF(mygo.PDFOptions{
					PageSize: size,
				})
				over()
				if err != nil {
					s.exportFailed("the page could not be printed", err)
					return
				}
				if err := os.WriteFile(dest, pdf, 0o644); err != nil {
					s.exportFailed("the PDF could not be written", err)
					return
				}
				s.say(locale.SayExportWritten(filepath.Base(dest)))
			}()
		})
		// A page the webview would not read is said, and the window let
		// go of, rather than waited on for a printing that never comes.
		offFailed = win.Page().OnDidFailLoad(func(e *mygo.LoadError) {
			offFailed()
			over()
			s.exportFailed("the page could not be printed", e)
		})
		if err := win.Page().LoadFile(name); err != nil {
			over()
			s.exportFailed("the page could not be printed", err)
		}
	})
}

// writePage is the page to be printed, put down beside the PDF it becomes
// under a name no one is meant to see. It is the caller's to take away
// once the printing is over.
func writePage(dir string, page []byte) (string, error) {
	f, err := os.CreateTemp(dir, ".xinkgo-*.html")
	if err != nil {
		return "", err
	}
	name := f.Name()
	if _, err := f.Write(page); err != nil {
		f.Close()
		os.Remove(name)
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}

// exportFailed is a note that would not be written out: said in the
// window, and in the log, which is where the reason goes.
func (s *appState) exportFailed(what string, err error) {
	slog.Warn(what, "err", err)
	s.say(locale.SayExportFailed())
}

// say is what the window says at the bottom, from whichever goroutine did
// the work: the window is drawn on the main thread, and it is the main
// thread that is to be told.
func (s *appState) say(text string) {
	mygo.RunOnMain(func() { s.ws.Say(text) })
}

// cssFont is the family a page is set in, as CSS names it. A family the
// machine keeps for its own — the one the interface is set in, which the
// configuration names as a kind and not as a family — is left to the
// browser to choose.
func cssFont(family string) string {
	if family == "" || strings.HasPrefix(family, ".") {
		return ""
	}
	return `"` + family + `", system-ui, -apple-system, sans-serif`
}
