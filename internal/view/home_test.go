package view

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/helightxu/xinkgo/internal/sysinfo"
)

// tester is a view under test, drawn as it would be in the window.
func tester(t *testing.T, h *Home) *ui.Tester {
	t.Helper()
	return ui.NewTester(h.View, 520, 400)
}

func testInfo() sysinfo.Info {
	return sysinfo.Current("Xinkgo", "0.1.0")
}

// TestHomeView_counts checks that the buttons change the state the window
// shows, and that the window then shows it.
func TestHomeView_counts(t *testing.T) {
	h := NewHome(testInfo())
	tt := tester(t, h)
	if err := tt.Click("Increment"); err != nil {
		t.Fatal(err)
	}
	if h.Count() != 1 || !tt.HasText("1") {
		t.Errorf("count %d after a click, texts %q", h.Count(), tt.Texts())
	}
	if err := tt.Click("Decrement"); err != nil {
		t.Fatal(err)
	}
	if h.Count() != 0 {
		t.Errorf("count %d after increment and decrement", h.Count())
	}
}

// TestHomeView_greetsByName checks that what the user types reaches the
// state and comes back as the greeting.
func TestHomeView_greetsByName(t *testing.T) {
	h := NewHome(testInfo())
	tt := tester(t, h)
	if err := tt.Click("Name"); err != nil {
		t.Fatal(err)
	}
	tt.Type("Ada")
	if !tt.HasText("Hello, Ada!") {
		t.Errorf("texts %q after typing a name", tt.Texts())
	}
}

// TestHomeView_showsVersion checks that the window says which version it
// is, so a user can tell builds apart.
func TestHomeView_showsVersion(t *testing.T) {
	tt := tester(t, NewHome(testInfo()))
	if !tt.HasText("version 0.1.0") {
		t.Errorf("texts %q without the version", tt.Texts())
	}
}

// TestHomeView_webButtonHiddenWithoutHandler checks that the window offers
// the web window only when the app can open one.
func TestHomeView_webButtonHiddenWithoutHandler(t *testing.T) {
	tt := tester(t, NewHome(testInfo()))
	if tt.HasText("Open a web window") {
		t.Error("the button is there although the app set no handler")
	}
}

// TestHomeView_webButtonCallsHandler checks that clicking the button calls
// the app.
func TestHomeView_webButtonCallsHandler(t *testing.T) {
	h := NewHome(testInfo())
	opened := 0
	h.OnWeb = func() { opened++ }
	tt := tester(t, h)
	if err := tt.Click("Open a web window"); err != nil {
		t.Fatal(err)
	}
	if opened != 1 {
		t.Errorf("handler called %d times, want 1", opened)
	}
}
