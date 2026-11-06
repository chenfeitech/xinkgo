package view

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
)

func TestPluginIcon_knownAndFallback(t *testing.T) {
	if pluginIcon("globe") != IconGlobe ||
		pluginIcon("calendar-days") != IconDailyNote ||
		pluginIcon("layout-template") != IconTemplate {
		t.Fatal("a built-in ribbon icon was not resolved")
	}
	if pluginIcon("unknown") != IconPluginRibbon {
		t.Error("an unknown icon did not fall back")
	}
}

// The three built-in plugins' ribbon buttons are drawn and clickable, so
// a plugin that is on really does have a shortcut in the tree's ribbon.
func TestPluginRibbon_coreIconsAreClickable(t *testing.T) {
	w := &Workspace{PluginRibbon: []PluginAction{
		{ID: "daily-notes:open-today", Name: "Open daily note",
			Icon: "calendar-days"},
		{ID: "templates:insert", Name: "Insert template",
			Icon: "layout-template"},
		{ID: "web-viewer:open", Name: "Open web viewer",
			Icon: "globe"},
	}}
	called := ""
	w.OnPluginAction = func(id string) { called = id }
	tt := ui.NewTester(w.pluginRibbon, 100, 200)
	for label, want := range map[string]string{
		"Open daily note": "daily-notes:open-today",
		"Insert template": "templates:insert",
		"Open web viewer": "web-viewer:open",
	} {
		if err := tt.Click(label); err != nil {
			t.Fatalf("click %q: %v", label, err)
		}
		if called != want {
			t.Errorf("click %q reported %q, want %q", label, called, want)
		}
		called = ""
	}
}

func TestCommandGroup(t *testing.T) {
	if got := commandGroup(&PluginCommand{ID: "x:y", Group: "Group"}); got != "Group" {
		t.Errorf("group = %q", got)
	}
	if got := commandGroup(&PluginCommand{ID: "x:y"}); got != "x:y" {
		t.Errorf("fallback = %q", got)
	}
}

func TestInsertInActive_putsTextInTheNote(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	w := NewWorkspace(config.Default())
	if _, err := w.Tabs.Create(path, "hello"); err != nil {
		t.Fatal(err)
	}
	if !w.InsertInActive(" world") {
		t.Fatal("there is an active note, insertion should work")
	}
	n := w.Tabs.ActiveNote()
	if n == nil || n.Note.Text != " worldhello" {
		t.Fatalf("text = %+v", n)
	}
	if w.ActiveBasename() != "note" {
		t.Errorf("basename = %q", w.ActiveBasename())
	}
}

func TestInsertInActive_noNoteIsFalse(t *testing.T) {
	w := NewWorkspace(config.Default())
	if w.InsertInActive("x") || w.ActiveBasename() != "" {
		t.Error("a window with no note reported an insertion")
	}
}

func TestPicker_reportsChoice(t *testing.T) {
	w := NewWorkspace(config.Default())
	got := ""
	w.OpenPicker("Pick", []string{"alpha", "beta"}, func(name string) {
		got = name
	})
	w.pickerState.query = "beta"
	w.choosePicker(0)
	if got != "beta" {
		t.Errorf("picked %q", got)
	}
	if w.pickerState.Open {
		t.Error("the picker stayed open after a choice")
	}
}

func TestPicker_openingAgainReplacesTheFirst(t *testing.T) {
	w := NewWorkspace(config.Default())
	w.OpenPicker("A", []string{"a"}, nil)
	w.OpenPicker("B", []string{"b"}, nil)
	if w.pickerState.Title != "B" ||
		len(w.pickerState.Items) != 1 ||
		w.pickerState.Items[0] != "b" {
		t.Errorf("picker state = %+v", w.pickerState)
	}
}

func TestNewWorkspace_coreIDs(t *testing.T) {
	// The registry and the view's gate constants must stay in step.
	if CoreDailyNotes != "daily-notes" ||
		CoreTemplates != "templates" ||
		CoreWebView != "web-viewer" {
		t.Fatal("a core plugin ID changed")
	}
}

// The preview pane's web tab strip: the fixed Preview tab, each page's
// tab, its close button and the new-tab button all report to the app.
func TestWebTabStrip_previewAndTabsAreClickable(t *testing.T) {
	w := NewWorkspace(config.Default())
	w.WebTabs = []WebTabInfo{
		{ID: "web-1", Title: "Example", Address: "https://example.com",
			Active: true},
		{ID: "web-2", Title: "Other", Address: "https://other.example"},
	}
	w.WebTabActive = "web-1"
	preview, selected, closed, opened := "", "", "", ""
	w.OnWebShowPreview = func() { preview = "yes" }
	w.OnWebSelectTab = func(id string) { selected = id }
	w.OnWebCloseTab = func(id string) { closed = id }
	w.OnWebNewTab = func() { opened = "yes" }
	tt := ui.NewTester(w.webTabStrip, 600, 80)
	for _, want := range []string{
		locale.WebTabPreview(), "Example", "Other",
	} {
		if !tt.HasText(want) {
			t.Errorf("the tab strip does not say %q", want)
		}
	}
	if err := tt.Click(locale.WebTabPreview()); err != nil {
		t.Fatal(err)
	}
	if preview != "yes" {
		t.Error("the Preview tab did not report a click")
	}
	if err := tt.Click("Example"); err != nil {
		t.Fatal(err)
	}
	if selected != "web-1" {
		t.Errorf("selected = %q", selected)
	}
	if err := tt.Click(locale.WebCloseTab()); err != nil {
		t.Fatal(err)
	}
	if closed != "web-1" {
		t.Errorf("closed = %q", closed)
	}
	if err := tt.Click("+"); err != nil {
		t.Fatal(err)
	}
	if opened != "yes" {
		t.Error("the new-tab button did not report a click")
	}
}

// A web tab with no native page still draws the address bar and its
// buttons, so a page that failed to load is not an empty pane.
func TestWebPane_toolbarIsDrawn(t *testing.T) {
	w := NewWorkspace(config.Default())
	w.WebTabs = []WebTabInfo{{
		ID: "web-1", Title: "Example", Active: true,
	}}
	w.WebTabActive = "web-1"
	w.WebStatus = locale.WebLoading()
	tt := ui.NewTester(w.webPane, 420, 300)
	for _, want := range []string{
		locale.WebBack(), locale.WebForward(), locale.WebReload(),
		locale.WebHome(), locale.WebReader(), locale.WebSave(),
		locale.WebExternal(),
	} {
		if !tt.HasText(want) {
			t.Errorf("the web toolbar does not say %q", want)
		}
	}
}

// A long page title must not widen its tab: the text is shortened before
// layout, the tab has a fixed width, and the close button and the next tab
// stay where they are and stay clickable.
func TestWebTab_longTitleIsBounded(t *testing.T) {
	long := "A very long web page title that would otherwise overflow its tab and cover the close button"
	w := NewWorkspace(config.Default())
	w.WebTabs = []WebTabInfo{
		{ID: "web-1", Title: long, Address: "https://example.com",
			Active: true},
		{ID: "web-2", Title: "Second", Address: "https://second.example"},
	}
	w.WebTabActive = "web-1"
	selected, closed := "", ""
	w.OnWebSelectTab = func(id string) { selected = id }
	w.OnWebCloseTab = func(id string) { closed = id }
	tt := ui.NewTester(w.webTabStrip, 600, 80)
	short := shortWebTitle(long)
	if !tt.HasText(short) {
		t.Fatalf("the shortened title %q is not drawn", short)
	}
	for _, text := range tt.Texts() {
		if strings.Contains(text, long) {
			t.Errorf("the full long title was drawn: %q", text)
		}
	}
	box, ok := tt.Find(short)
	if !ok {
		t.Fatal("the shortened title has no box")
	}
	// Find hits the tab's own label, so this measures the tab, not the
	// text: a long title must never make the tab wider than its fixed
	// width and push its close button and the next tab aside.
	if box.W > webTabWidth+1 {
		t.Errorf("tab grew to %.1f DIP, fixed width is %.1f",
			box.W, webTabWidth)
	}
	if err := tt.Click(short); err != nil {
		t.Fatal(err)
	}
	if selected != "web-1" {
		t.Errorf("selecting the tab reported %q", selected)
	}
	if err := tt.Click(locale.WebCloseTab()); err != nil {
		t.Fatal(err)
	}
	if closed != "web-1" {
		t.Errorf("closing the tab reported %q", closed)
	}
	if !tt.HasText("Second") {
		t.Error("the tab after the long one was pushed out of view")
	}
}

func TestShortWebTitle_boundsByRunes(t *testing.T) {
	short := shortWebTitle(strings.Repeat("长", 40))
	if got := len([]rune(short)); got != webTabTitleRunes {
		t.Errorf("CJK title has %d runes, want %d", got,
			webTabTitleRunes)
	}
	if !strings.HasSuffix(short, "…") {
		t.Errorf("a cut title has no ellipsis: %q", short)
	}
	if shortWebTitle("Short") != "Short" ||
		shortWebTitle("  ") == "" {
		t.Error("a short or empty title was changed")
	}
}
