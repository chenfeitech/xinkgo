package app

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/snapshot"
	"github.com/helightxu/xinkgo/internal/theme"
)

// keptOne is a snapshot as the window shows it: where it stands, what it
// was taken of, when, and enough of it to tell which one it is.
type keptOne struct {
	Path    string
	Name    string
	When    string
	Preview string
}

// openHistory shows the window of what was kept before notes were changed,
// or brings it back when it is already open: there is one of them.
func (s *appState) openHistory() {
	if s.histWin != nil && !s.histWin.IsDestroyed() {
		s.histWin.Show()
		s.histWin.Focus()
		return
	}
	s.histStatus = ""
	s.loadKept()
	s.histWin = mygo.NewWindow(mygo.WindowOptions{
		Title:     "Snapshots",
		Width:     620,
		Height:    480,
		MinWidth:  420,
		MinHeight: 300,
		// The window shows the interface MyGo draws, not a web page.
		Content: ui.View(s.historyView),
	})
}

// historyView is the window of what was kept: one line for each snapshot,
// the newest first, and the way to have it put back.
func (s *appState) historyView(c *ui.Context) {
	theme.Apply(c, s.config.Basic.Theme)
	dir := snapshot.Dir(s.dir)
	ui.Scroll(c).TrackScroll(&s.histScroll).Fill().Children(func() {
		ui.Column(c).Fill().Gap(8).Padding(8).Children(func() {
			ui.Text(c, "Snapshots").Bold()
			ui.Text(c, "What a note said before it was changed. The "+
				"oldest "+strconv.Itoa(snapshot.Max)+" are kept; "+
				"putting one "+
				"back goes into the note of that name in the vault "+
				"that is open, and what stands there now is kept "+
				"first.").TextColor(c.Theme().TextMuted)
			if len(s.kept) == 0 {
				ui.Text(c, "Nothing kept yet: a note is kept when a "+
					"revision is written into it, or when one is put "+
					"back.").TextColor(c.Theme().TextMuted)
				return
			}
			ui.Text(c, strconv.Itoa(snapshot.Count(dir))+" kept").
				TextColor(c.Theme().TextMuted)
			for _, one := range s.kept {
				s.historyItem(c, one)
			}
			if s.histStatus != "" {
				ui.Text(c, s.histStatus)
			}
		})
	})
}

// historyItem is one snapshot: what it was taken of, when, the beginning
// of it, and the way to have it put back.
func (s *appState) historyItem(c *ui.Context, one keptOne) {
	ui.Column(c).Gap(2).Children(func() {
		ui.Row(c).Gap(6).Children(func() {
			ui.Text(c, one.Name).Grow(1)
			ui.Text(c, one.When).FontSize(11).
				TextColor(c.Theme().TextMuted)
			if ui.Button(c.Key("hist-back-"+filepath.Base(one.Path)),
				"Put it back").Clicked() {
				s.putBack(one.Path)
			}
		})
		ui.Text(c, one.Preview).FontSize(11).
			TextColor(c.Theme().TextMuted)
	})
}

// loadKept reads what was kept, so that the window shows it without the
// disk being read again every frame.
func (s *appState) loadKept() {
	all := snapshot.List(snapshot.Dir(s.dir))
	s.kept = make([]keptOne, 0, len(all))
	for _, path := range all {
		text := ""
		if data, err := os.ReadFile(path); err == nil {
			text = string(data)
		}
		s.kept = append(s.kept, keptOne{
			Path:    path,
			Name:    snapshot.Original(path),
			When:    whenTaken(path),
			Preview: firstWords(text),
		})
	}
}

// putBack is a snapshot put back into the note it was taken of.
//
// What the note says now is kept as a snapshot first: the way back is
// itself a thing one can come back from, and putting words back is not a
// thing that throws away what stood there.
func (s *appState) putBack(path string) {
	name := snapshot.Original(path)
	where, err := s.whereBack(name)
	if err != nil {
		s.histStatus = err.Error()
		return
	}
	was, err := os.ReadFile(path)
	if err != nil {
		s.histStatus = "that snapshot could not be read"
		slog.Warn("a snapshot could not be read", "path", path,
			"err", err)
		return
	}
	now, err := s.readNote(where)
	if err != nil {
		s.histStatus = "the note could not be read"
		slog.Warn("a note could not be read", "path", where, "err", err)
		return
	}
	// What stands there now is kept before it is written over, so that
	// putting back is not itself a change one cannot come back from.
	s.snapshotNote(where, now)
	if err := s.writeNote(where, string(was)); err != nil {
		s.histStatus = "the note could not be written"
		slog.Warn("a note could not be written", "path", where,
			"err", err)
		return
	}
	if s.idx != nil {
		s.idx.Put(where, string(was))
	}
	// The note that is open is made to show it, or its next save would
	// write the words it had before over the ones put back.
	if s.ws != nil && s.ws.Tabs != nil {
		s.ws.Tabs.PutText(where)
	}
	s.histStatus = name + " is as it was again"
	s.loadKept()
	slog.Info("snapshot put back", "from", path, "into", where)
}

// whereBack is which note of the vault a snapshot of that name goes back
// into: the note of that name that is open, and otherwise the one the
// vault has.
//
// When the vault has more than one of the name, or none at all, that is
// said rather than guessed at: words put back into the wrong note are
// worse than words not put back at all.
func (s *appState) whereBack(name string) (string, error) {
	if s.ws != nil && s.ws.Tabs != nil {
		for _, path := range s.ws.Tabs.Paths() {
			if filepath.Base(path) == name {
				return path, nil
			}
		}
	}
	found := []string{}
	if s.idx != nil {
		for _, path := range s.idx.Paths() {
			if filepath.Base(path) == name {
				found = append(found, path)
			}
		}
	}
	switch len(found) {
	case 0:
		return "", fmt.Errorf("%s is not in this vault", name)
	case 1:
		return found[0], nil
	}
	return "", fmt.Errorf("more than one note is called %s", name)
}

// whenTaken is when a snapshot was taken, as the user's clock says it.
func whenTaken(path string) string {
	when, ok := snapshot.Taken(path)
	if !ok {
		return ""
	}
	return when.Local().Format("2006-01-02 15:04")
}

// firstWords is the beginning of a snapshot, so that one can tell which
// it is without opening it.
func firstWords(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		runes := []rune(line)
		if len(runes) > 80 {
			return string(runes[:80]) + "…"
		}
		return line
	}
	return "(nothing in it)"
}
