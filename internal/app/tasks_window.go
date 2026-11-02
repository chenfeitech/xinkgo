package app

import (
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/note"
	"github.com/helightxu/xinkgo/internal/task"
)

// taskDraft is the task being put together: which kind of work it is, and
// the tags it is about. It is kept apart from the task itself until it is
// planned, for a draft is not a thing that has been worked out.
type taskDraft struct {
	Kind   string
	Target string
	NewTag string
}

// openTasks shows the window of the long work over the vault, or brings
// it back when it is already open: there is one of them.
func (s *appState) openTasks() {
	if s.tasksWin != nil && !s.tasksWin.IsDestroyed() {
		s.tasksWin.Show()
		s.tasksWin.Focus()
		return
	}
	s.tasksWin = mygo.NewWindow(mygo.WindowOptions{
		Title:     locale.TitleLibraryTasks(),
		Width:     680,
		Height:    520,
		MinWidth:  480,
		MinHeight: 320,
		// The window shows the interface MyGo draws, not a web page.
		Content: ui.View(s.tasksView),
	})
}

// tasksView is the window of the long work: what is to be done, what it
// would come to, and the way to have it done a batch at a time.
func (s *appState) tasksView(c *ui.Context) {
	ui.Scroll(c).TrackScroll(&s.taskScroll).Fill().Children(func() {
		ui.Column(c).Fill().Gap(8).Padding(8).Children(func() {
			s.taskBeing(c)
			if s.work == nil {
				return
			}
			s.taskPlan(c)
		})
	})
}

// taskBeing is the task put together: which kind of work, and about which
// tag. A new name is only asked for when a tag is being changed.
func (s *appState) taskBeing(c *ui.Context) {
	ui.Column(c).Gap(4).Children(func() {
		ui.Text(c, locale.TasksNew()).Bold()
		ui.Row(c).Gap(6).Children(func() {
			ui.Text(c, locale.TasksWhat()).Grow(1)
			kinds := []string{
				task.Retag.ID(), task.AddTag.ID(), task.RemoveTag.ID(),
			}
			kind := kindOf(s.taskDraft.Kind).Label()
			ui.Select(c, &kind, taskKindLabels(kinds))
			s.taskDraft.Kind = kindIDOf(kind, kinds)
		})
		ui.Row(c).Gap(6).Children(func() {
			ui.Text(c, locale.TasksTag()).Grow(1)
			ui.TextInput(c, &s.taskDraft.Target).
				Placeholder(locale.TasksTagPlaceholder())
		})
		if s.taskDraft.Kind == task.Retag.ID() {
			ui.Row(c).Gap(6).Children(func() {
				ui.Text(c, locale.TasksInto()).Grow(1)
				ui.TextInput(c, &s.taskDraft.NewTag).
					Placeholder(locale.TasksIntoPlaceholder())
			})
		}
		ui.Row(c).Gap(6).Children(func() {
			if ui.Button(c.Key("task-plan"), locale.TasksPlan()).
				Label(locale.TasksPlanLabel()).
				Clicked() {
				s.planTask()
			}
			if ui.Button(c.Key("task-drop"), locale.TasksDrop()).Clicked() {
				s.work = nil
			}
		})
	})
}

// taskPlan is the task as it stands: what it set out to do, what came of
// each note, and the way to have the next batch done.
func (s *appState) taskPlan(c *ui.Context) {
	one := s.work
	done, skipped, failed := one.Stats()
	ui.Column(c).Gap(4).Children(func() {
		ui.Text(c, one.Title()).Bold()
		ui.Text(c, locale.TaskStats(one.Status.Label(),
			strconv.Itoa(done), strconv.Itoa(skipped),
			strconv.Itoa(failed),
			strconv.Itoa(one.Pending()))).
			TextColor(c.Theme().TextMuted)
		ui.Row(c).Gap(6).Children(func() {
			if !one.Finished() {
				if ui.Button(c.Key("task-run"), locale.TasksRun()).
					Label(locale.TasksRunLabel(
						strconv.Itoa(task.BatchSize))).Clicked() {
					s.runTask()
				}
			}
		})
		for i := range one.Items {
			s.taskItem(c, &one.Items[i])
		}
	})
}

// taskItem is one note of the task: where it is, what came of it, and the
// lines that would change.
func (s *appState) taskItem(c *ui.Context, item *task.TaskItem) {
	ui.Column(c).Gap(2).Children(func() {
		ui.Row(c).Gap(6).Children(func() {
			ui.Text(c, shortPath(s.root, item.Path)).Grow(1)
			ui.Text(c, item.State.Label()).FontSize(11).
				TextColor(c.Theme().TextMuted)
		})
		for _, edit := range item.Preview {
			ui.Text(c, "− "+edit.Before).FontSize(11).
				TextColor(c.Theme().TextMuted)
			ui.Text(c, "+ "+edit.After).FontSize(11)
		}
	})
}

// planTask is the task worked out over the whole vault: every note read
// once, and what would change in it said before anything is written.
func (s *appState) planTask() {
	kind := kindOf(s.taskDraft.Kind)
	if s.taskDraft.Target == "" {
		return
	}
	// A tag changed into nothing is not a thing to ask the vault for.
	if kind == task.Retag && s.taskDraft.NewTag == "" {
		return
	}
	paths := []string{}
	if s.idx != nil {
		paths = s.idx.Paths()
	}
	one := task.Plan("t", kind, s.taskDraft.Target, s.taskDraft.NewTag,
		paths, s.readNote)
	s.work = &one
	slog.Info("task planned", "kind", kind.ID(),
		"target", s.taskDraft.Target, "notes", len(one.Items))
}

// runTask is the next batch of the task done, and the vault indexed again
// for what changed in it.
func (s *appState) runTask() {
	if s.work == nil {
		return
	}
	did := task.Run(s.work, task.BatchSize, s.readNote, s.writeNote)
	// What was written is not what the index holds any more.
	if s.idx != nil {
		for i := range s.work.Items {
			item := &s.work.Items[i]
			if item.State.Kind != task.ItemDone {
				continue
			}
			if text, err := s.readNote(item.Path); err == nil {
				s.idx.Put(item.Path, text)
			}
		}
	}
	slog.Info("task ran", "did", did, "left", s.work.Pending())
}

// readNote is a note's text off the disk: the disk is the truth of a
// note, and a plan may be older than it.
func (s *appState) readNote(path string) (string, error) {
	if !withinRoot(s.root, path) {
		return "", os.ErrPermission
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// writeNote is a note written back, whole or not at all: a task stops
// halfway often enough that a half-written note is a real thing to guard
// against.
func (s *appState) writeNote(path, text string) error {
	if !withinRoot(s.root, path) {
		return os.ErrPermission
	}
	one, err := note.Open(path)
	if err != nil {
		return err
	}
	one.SetText(text)
	return one.Save()
}

// withinRoot is whether the path stands inside the vault. A task works on
// the vault, and on nothing else.
func withinRoot(root, path string) bool {
	if root == "" {
		return false
	}
	rel, err := filepath.Rel(root, filepath.Clean(path))
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(
		filepath.Separator))
}

// taskKindLabels is what each kind of work is called in the language the
// app shows, and kindIDOf the word the configuration keeps for one of
// them: the row shows the names, and the file keeps the words.
func taskKindLabels(kinds []string) []string {
	out := make([]string, len(kinds))
	for i, id := range kinds {
		out[i] = kindOf(id).Label()
	}
	return out
}

func kindIDOf(label string, kinds []string) string {
	for _, id := range kinds {
		if kindOf(id).Label() == label {
			return id
		}
	}
	return kinds[0]
}

// kindOf is the kind named by that word.
func kindOf(id string) task.Kind {
	switch id {
	case task.AddTag.ID():
		return task.AddTag
	case task.RemoveTag.ID():
		return task.RemoveTag
	}
	return task.Retag
}

// shortPath is a path as the vault knows it: from the root, which is the
// shorter thing to say.
func shortPath(root, path string) string {
	if root == "" {
		return path
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}
