package app

import (
	"strings"
	"testing"
	"time"

	"github.com/helightxu/xinkgo/internal/ai"
	"github.com/helightxu/xinkgo/internal/config"
)

// TestNewModel_everyNewModelHasANameOfItsOwn checks that adding a model
// never makes two that are known by the same name: the name is what the
// app asks by.
func TestNewModel_everyNewModelHasANameOfItsOwn(t *testing.T) {
	s := &appState{}
	s.draft.cfg = config.Default()
	seen := map[string]bool{}
	for range 4 {
		m := s.newModel()
		if seen[m.ID] {
			t.Fatalf("%q was given out twice", m.ID)
		}
		seen[m.ID] = true
		s.draft.cfg.Ai.Models = append(s.draft.cfg.Ai.Models, m)
	}
}

// TestNewModel_oneThatIsNotSetUpYetIsStillUsable checks that a model just
// added is turned on and asks the local server, so that filling in an
// address is all that is left to do.
func TestNewModel_oneThatIsNotSetUpYetIsStillUsable(t *testing.T) {
	m := (&appState{}).newModel()
	if !m.Enabled {
		t.Fatal("a new model is turned off")
	}
	if m.Kind != config.ModelOllama {
		t.Fatalf("kind = %q, want the local one", m.Kind)
	}
	if m.Params.NumPredict == 0 || m.Params.Temperature == 0 {
		t.Fatalf("params = %+v, want the defaults", m.Params)
	}
	if m.ID == "" || m.Label == "" {
		t.Fatalf("a new model has no name (%q)", m.ID)
	}
}

// TestForgetModel_itIsAlsoTakenOutOfTheTwoPlacesItWasNamed checks that a
// model that is gone is not left as the one to ask, or the one to ask
// when that one cannot answer.
func TestForgetModel_itIsAlsoTakenOutOfTheTwoPlacesItWasNamed(t *testing.T) {
	st := &settings{cfg: config.Default()}
	st.cfg.Ai.Models = []config.ModelEntry{
		{ID: "first", Enabled: true},
		{ID: "second", Enabled: true},
	}
	st.cfg.Ai.DefaultModel = "first"
	st.cfg.Ai.FallbackModel = "first"
	_ = st.stateOf("first")

	st.forgetModel("first")
	if len(st.cfg.Ai.Models) != 1 || st.cfg.Ai.Models[0].ID != "second" {
		t.Fatalf("models = %v, want only the second", st.cfg.Ai.Models)
	}
	if st.cfg.Ai.DefaultModel != "" || st.cfg.Ai.FallbackModel != "" {
		t.Fatalf("default = %q fallback = %q, want both cleared",
			st.cfg.Ai.DefaultModel, st.cfg.Ai.FallbackModel)
	}
	if _, there := st.models["first"]; there {
		t.Fatal("the row of a forgotten model was kept")
	}
}

// TestForgetModel_anotherModelIsLeftAlone checks that forgetting one does
// not touch what was chosen for another.
func TestForgetModel_anotherModelIsLeftAlone(t *testing.T) {
	st := &settings{cfg: config.Default()}
	st.cfg.Ai.Models = []config.ModelEntry{
		{ID: "first"}, {ID: "second"},
	}
	st.cfg.Ai.DefaultModel = "second"
	st.forgetModel("first")
	if st.cfg.Ai.DefaultModel != "second" {
		t.Fatalf("default = %q, want it left alone",
			st.cfg.Ai.DefaultModel)
	}
}

// TestSaidOf_whatTryingAModelCameTo checks that what the probe found is
// said in a line a person can read.
func TestSaidOf_whatTryingAModelCameTo(t *testing.T) {
	s := &appState{}
	got := s.saidOf(&ai.Probe{OK: true, Tools: true,
		Took: 120 * time.Millisecond})
	if !strings.Contains(got, "120") {
		t.Fatalf("said %q, want how long it took", got)
	}
	if !strings.Contains(got, "tools") {
		t.Fatalf("said %q, want it said that it calls tools", got)
	}
	got = s.saidOf(&ai.Probe{OK: true, Took: time.Millisecond})
	if strings.Contains(got, "and it calls tools") {
		t.Fatalf("said %q, want it said that it calls none", got)
	}
	got = s.saidOf(&ai.Probe{Err: ai.ErrTimeout})
	if !strings.Contains(got, ai.ErrTimeout.Error()) {
		t.Fatalf("said %q, want what went wrong", got)
	}
}

// TestSaidOf_aFailureWithNothingToSayIsStillSaid checks that a probe that
// failed without knowing why is not shown as one that answered.
func TestSaidOf_aFailureWithNothingToSayIsStillSaid(t *testing.T) {
	if got := (&appState{}).saidOf(&ai.Probe{}); got == "" {
		t.Fatal("nothing was said of a model that could not be reached")
	}
}

// TestHolds_whetherAListHoldsIt checks the one thing the capability boxes
// ask all day.
func TestHolds_whetherAListHoldsIt(t *testing.T) {
	if !holds([]string{"a", "b"}, "b") {
		t.Fatal("b was not found")
	}
	if holds([]string{"a"}, "b") {
		t.Fatal("b was found although it is not there")
	}
	if holds(nil, "b") {
		t.Fatal("b was found in nothing")
	}
}

// TestStateOf_theSameRowComesBack checks that what a model's row holds
// between two frames is the same thing and not a new one each time.
func TestStateOf_theSameRowComesBack(t *testing.T) {
	st := &settings{}
	first := st.stateOf("qwen")
	first.open = true
	if st.stateOf("qwen") != first {
		t.Fatal("the row was made again")
	}
	if st.stateOf("other") == first {
		t.Fatal("two models share one row")
	}
}

// modelsOf is an app whose settings hold those models.
func modelsOf(ids ...string) *appState {
	s := &appState{}
	s.draft.cfg = config.Default()
	s.draft.cfg.Ai.Models = nil
	for _, id := range ids {
		s.draft.cfg.Ai.Models = append(s.draft.cfg.Ai.Models,
			config.ModelEntry{ID: id, Label: id, Enabled: true})
	}
	return s
}

// TestCloneModel_theCopyStandsByTheOneItWasTakenOf checks that a copy is
// put next to the model it came from, and is the same model otherwise.
func TestCloneModel_theCopyStandsByTheOneItWasTakenOf(t *testing.T) {
	s := modelsOf("first", "second")
	s.cloneModel("first")
	list := s.draft.cfg.Ai.Models
	if len(list) != 3 {
		t.Fatalf("models = %d, want three", len(list))
	}
	if list[1].ID != "first-copy" {
		t.Fatalf("the copy is at %q, want it next to its own",
			list[1].ID)
	}
	if list[0].ID != "first" || list[2].ID != "second" {
		t.Fatalf("order = %q, want the rest left where they were",
			[]string{list[0].ID, list[2].ID})
	}
	if list[1].Label != "first-copy" {
		t.Fatalf("label = %q, want it not to wear the other's name",
			list[1].Label)
	}
}

// TestCloneModel_twoCopiesOfOneAreToldApart checks that a copy made
// twice is given two names: the name is what the app asks by.
func TestCloneModel_twoCopiesOfOneAreToldApart(t *testing.T) {
	s := modelsOf("first")
	s.cloneModel("first")
	s.cloneModel("first")
	ids := []string{}
	for _, m := range s.draft.cfg.Ai.Models {
		ids = append(ids, m.ID)
	}
	// Each copy stands next to the one it was taken of, so the last
	// made is the nearer of the two.
	if len(ids) != 3 || ids[1] != "first-copy2" ||
		ids[2] != "first-copy" {
		t.Fatalf("ids = %v, want two copies told apart", ids)
	}
}

// TestCloneModel_theCopyIsChangedWithoutTheOtherChanging checks the one
// thing a copy is for: it can be changed afterwards.
func TestCloneModel_theCopyIsChangedWithoutTheOtherChanging(t *testing.T) {
	s := modelsOf("first")
	s.draft.cfg.Ai.Models[0].Capabilities = []string{"tools"}
	s.draft.cfg.Ai.Models[0].BaseURL = "http://localhost:11434"
	s.cloneModel("first")
	one, two := &s.draft.cfg.Ai.Models[0], &s.draft.cfg.Ai.Models[1]
	two.Capabilities[0] = "vision"
	two.BaseURL = "http://elsewhere:1234"
	if one.Capabilities[0] != "tools" {
		t.Fatalf("the one it came from can %q now",
			one.Capabilities[0])
	}
	if one.BaseURL == two.BaseURL {
		t.Fatal("the copy asks the same place as the original")
	}
}

// TestMoveModel_earlierAndLater checks that a model may be moved through
// the list, and not out of either end of it.
func TestMoveModel_earlierAndLater(t *testing.T) {
	s := modelsOf("a", "b", "c")
	s.moveModel("c", -1)
	if idsOf(s) != "a,c,b" {
		t.Fatalf("order = %s, want c moved earlier", idsOf(s))
	}
	s.moveModel("a", 1)
	if idsOf(s) != "c,a,b" {
		t.Fatalf("order = %s, want a moved later", idsOf(s))
	}
	// The ends hold: nothing is moved out of the list.
	s.moveModel("c", -1)
	s.moveModel("b", 1)
	if idsOf(s) != "c,a,b" {
		t.Fatalf("order = %s, want the ends to hold", idsOf(s))
	}
	// A model that is not one of them is not a failure.
	s.moveModel("nope", -1)
	if idsOf(s) != "c,a,b" {
		t.Fatalf("order = %s, want it left alone", idsOf(s))
	}
}

// TestCloneModel_oneThatIsNotThereIsNotCopied checks that a copy asked
// for twice in one frame does not make two.
func TestCloneModel_oneThatIsNotThereIsNotCopied(t *testing.T) {
	s := modelsOf("a")
	s.cloneModel("nope")
	if len(s.draft.cfg.Ai.Models) != 1 {
		t.Fatalf("models = %d, want it left alone",
			len(s.draft.cfg.Ai.Models))
	}
}

// idsOf is the models' names, in the order they stand.
func idsOf(s *appState) string {
	ids := []string{}
	for _, m := range s.draft.cfg.Ai.Models {
		ids = append(ids, m.ID)
	}
	return strings.Join(ids, ",")
}
