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
// added is turned on and is one of the user's own, so that filling in an
// address and a key is all that is left to do: the models the machine
// holds are found by asking the machine, and are not written in.
func TestNewModel_oneThatIsNotSetUpYetIsStillUsable(t *testing.T) {
	m := (&appState{}).newModel()
	if !m.Enabled {
		t.Fatal("a new model is turned off")
	}
	if m.Source != config.ModelSourceCustom {
		t.Fatalf("source = %q, want one written in by hand", m.Source)
	}
	if m.Kind != config.ModelOpenAICompatible {
		t.Fatalf("kind = %q, want one spoken to as the clouds are",
			m.Kind)
	}
	if m.ApiKeySource != config.KeyPlain {
		t.Fatalf("key source = %q, want one written out",
			m.ApiKeySource)
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

// TestSettings_whatTheWindowShowsIsWhatTakesEffect checks the round trip
// of the knobs of asking: a window that shows one number and sets
// another is worse than a window with no knob at all.
func TestSettings_whatTheWindowShowsIsWhatTakesEffect(t *testing.T) {
	st := &settings{cfg: config.Default()}
	st.read(&st.cfg)
	if st.timeout != 60 || st.firstByte != 30 || st.tries != 3 ||
		st.backoff != 500 || st.steps != 8 || st.ctxChars != 12_000 ||
		st.toolChars != 4_000 {
		t.Fatalf("read = %v %v %v %v %v %v %v, want the defaults",
			st.timeout, st.firstByte, st.tries, st.backoff, st.steps,
			st.ctxChars, st.toolChars)
	}
	// They count in whole numbers: the nearest one is the one meant.
	st.timeout = 89.6
	st.tries = 2
	st.ctxChars = 20_000
	st.write(&st.cfg)
	if st.cfg.Ai.TimeoutSecs != 90 {
		t.Fatalf("timeout = %d, want 90", st.cfg.Ai.TimeoutSecs)
	}
	if st.cfg.Ai.Retry.MaxAttempts != 2 {
		t.Fatalf("tries = %d, want 2", st.cfg.Ai.Retry.MaxAttempts)
	}
	if st.cfg.Ai.MaxContextChars != 20_000 {
		t.Fatalf("context = %d, want 20000", st.cfg.Ai.MaxContextChars)
	}
	// What the window offers is what the settings keep: nothing here
	// is clamped away behind the user's back.
	st.cfg.Sanitize()
	if st.cfg.Ai.TimeoutSecs != 90 || st.cfg.Ai.MaxContextChars !=
		20_000 {
		t.Fatalf("sanitized = %d %d, want them kept",
			st.cfg.Ai.TimeoutSecs, st.cfg.Ai.MaxContextChars)
	}
}

// TestWhole_theNearestOneIsMeant checks that a number between two whole
// ones is not quietly brought down to the one below it.
func TestWhole_theNearestOneIsMeant(t *testing.T) {
	cases := map[float64]uint64{
		89.6: 90,
		89.4: 89,
		0.4:  0,
		2.5:  3,
	}
	for in, want := range cases {
		if got := whole(in); got != want {
			t.Errorf("whole(%v) = %d, want %d", in, got, want)
		}
	}
}

// ollamaFound is a model the machine holds, as asking it gives one back.
func ollamaFound(name string) ai.OllamaModel {
	return ai.OllamaModel{
		Name:          name,
		Size:          522928832,
		Family:        "qwen3",
		ParameterSize: "0.6B",
		ContextLimit:  32768,
		Capabilities:  []string{ai.CapCompletion, ai.CapTools},
	}
}

// TestAddOllamaModel_oneTheMachineHoldsIsTakenInAsTheMachineSaid checks
// the whole of what asking the machine is for: the model comes into the
// settings as one of the machine's, with what the machine said it can do,
// and with no address of its own to go out of step with the one above.
func TestAddOllamaModel_oneTheMachineHoldsIsTakenInAsTheMachineSaid(t *testing.T) {
	st := &settings{cfg: config.Default()}
	st.cfg.Ai.Models = nil
	st.addOllamaModel(ollamaFound("qwen3:0.6b"))
	if len(st.cfg.Ai.Models) != 1 {
		t.Fatalf("models = %d, want one", len(st.cfg.Ai.Models))
	}
	m := st.cfg.Ai.Models[0]
	if m.ID != "qwen3:0.6b" || m.Label != "qwen3:0.6b" {
		t.Fatalf("id = %q label = %q, want the machine's name",
			m.ID, m.Label)
	}
	if m.Source != config.ModelSourceOllama || !m.FromOllama() {
		t.Fatalf("source = %q, want one the machine holds", m.Source)
	}
	if m.Kind != config.ModelOllama {
		t.Fatalf("kind = %q, want the local one", m.Kind)
	}
	if m.BaseURL != "" {
		t.Fatalf("address = %q, want it left to the one above",
			m.BaseURL)
	}
	if !m.Enabled || !m.ToolsEnabled {
		t.Fatalf("enabled = %v tools = %v, want both on",
			m.Enabled, m.ToolsEnabled)
	}
	if m.ContextLimit == nil || *m.ContextLimit != 32768 {
		t.Fatalf("context = %v, want 32768", m.ContextLimit)
	}
	want := []string{ai.CapCompletion, ai.CapTools}
	if strings.Join(m.Capabilities, ",") != strings.Join(want, ",") {
		t.Fatalf("capabilities = %v, want %v", m.Capabilities, want)
	}
	// Ticking it is what put it there, and it stays ticked.
	if on, there := st.ollamaOn("qwen3:0.6b"); !on || !there {
		t.Fatalf("on = %v there = %v, want it asked", on, there)
	}
}

// TestAddOllamaModel_askingTwiceDoesNotMakeTwo checks that a model pulled
// again, or ticked twice, is the same model and not two of them: the name
// is what the app asks by.
func TestAddOllamaModel_askingTwiceDoesNotMakeTwo(t *testing.T) {
	st := &settings{cfg: config.Default()}
	st.cfg.Ai.Models = nil
	one := ollamaFound("qwen3:0.6b")
	st.addOllamaModel(one)
	one.Capabilities = []string{ai.CapCompletion, ai.CapThinking}
	st.addOllamaModel(one)
	if len(st.cfg.Ai.Models) != 1 {
		t.Fatalf("models = %d, want one", len(st.cfg.Ai.Models))
	}
	// What the machine says now is believed over what it said before.
	if holds(st.cfg.Ai.Models[0].Capabilities, ai.CapTools) {
		t.Fatalf("capabilities = %v, want the newer word kept",
			st.cfg.Ai.Models[0].Capabilities)
	}
}

// TestAddOllamaModel_whatTheHandChoseIsKept checks that capabilities the
// user ticked by hand are not overwritten by what the machine says: the
// hand was laid on them, and that is what is believed.
func TestAddOllamaModel_whatTheHandChoseIsKept(t *testing.T) {
	st := &settings{cfg: config.Default()}
	st.cfg.Ai.Models = []config.ModelEntry{{
		ID: "qwen3:0.6b", Source: config.ModelSourceOllama,
		Kind: config.ModelOllama, Enabled: false,
		Capabilities:       []string{ai.CapVision},
		CapabilitiesPinned: true,
	}}
	st.addOllamaModel(ollamaFound("qwen3:0.6b"))
	m := st.cfg.Ai.Models[0]
	if !m.Enabled {
		t.Fatal("it was not turned on")
	}
	if strings.Join(m.Capabilities, ",") != ai.CapVision {
		t.Fatalf("capabilities = %v, want the hand's kept",
			m.Capabilities)
	}
}

// TestOllamaOn_aModelOfTheSameNameWrittenByHandIsNotTheMachines checks that
// the two kinds are told apart by more than a name: a model the user wrote
// in does not answer to the machine's tick, and is not taken out by it.
func TestOllamaOn_aModelOfTheSameNameWrittenByHandIsNotTheMachines(t *testing.T) {
	st := &settings{cfg: config.Default()}
	st.cfg.Ai.Models = []config.ModelEntry{{
		ID: "qwen3:0.6b", Source: config.ModelSourceCustom,
		Kind: config.ModelOllama, Enabled: true,
	}}
	if on, there := st.ollamaOn("qwen3:0.6b"); on || there {
		t.Fatalf("on = %v there = %v, want it left to the user",
			on, there)
	}
}

// TestRenameModel_itIsFollowedEverywhereItWasNamed checks the one thing
// that makes renaming safe: a model asked by another name is still the
// one asked, wherever it had been named.
func TestRenameModel_itIsFollowedEverywhereItWasNamed(t *testing.T) {
	st := &settings{cfg: config.Default()}
	other := "qwen3:0.6b"
	st.cfg.Ai.Models = []config.ModelEntry{{
		ID: "model", Label: "model", Enabled: true,
	}}
	st.cfg.Ai.DefaultModel = "model"
	st.cfg.Ai.FallbackModel = "model"
	st.cfg.Ai.Route.Chat = &other
	st.cfg.Ai.Actions = []config.AiAction{{ID: "polish", Model: &other}}
	// A route of another model is not one of this one's.
	_ = st.stateOf("model")

	if !st.renameModel("model", "gpt-4o-mini") {
		t.Fatal("the new name was not taken")
	}
	if st.cfg.Ai.Models[0].ID != "gpt-4o-mini" {
		t.Fatalf("id = %q, want the new name", st.cfg.Ai.Models[0].ID)
	}
	if st.cfg.Ai.Models[0].Label != "gpt-4o-mini" {
		t.Fatalf("label = %q, want it to follow the name",
			st.cfg.Ai.Models[0].Label)
	}
	if st.cfg.Ai.DefaultModel != "gpt-4o-mini" {
		t.Fatalf("default = %q, want it to follow",
			st.cfg.Ai.DefaultModel)
	}
	if st.cfg.Ai.FallbackModel != "gpt-4o-mini" {
		t.Fatalf("fallback = %q, want it to follow",
			st.cfg.Ai.FallbackModel)
	}
	// What the row kept between two frames comes too.
	if _, there := st.models["model"]; there {
		t.Fatal("the row was left under the old name")
	}
	if _, there := st.models["gpt-4o-mini"]; !there {
		t.Fatal("the row was not kept under the new name")
	}
}

// TestRenameModel_aNameAnotherAnswersToIsNotTaken checks that two models
// are not made into one by name: a name another model already answers to
// is refused, and the model keeps its own.
func TestRenameModel_aNameAnotherAnswersToIsNotTaken(t *testing.T) {
	st := &settings{cfg: config.Default()}
	st.cfg.Ai.Models = []config.ModelEntry{
		{ID: "one", Enabled: true}, {ID: "two", Enabled: true},
	}
	if st.renameModel("one", "two") {
		t.Fatal("a name another answers to was taken")
	}
	if st.cfg.Ai.Models[0].ID != "one" || st.cfg.Ai.Models[1].ID != "two" {
		t.Fatalf("ids = %q %q, want them left alone",
			st.cfg.Ai.Models[0].ID, st.cfg.Ai.Models[1].ID)
	}
	if st.renameModel("one", "") {
		t.Fatal("no name at all was taken")
	}
}

// TestSizeOf_howMuchRoomAModelTakes checks that a number of bytes is said
// in words a person can read.
func TestSizeOf_howMuchRoomAModelTakes(t *testing.T) {
	cases := map[uint64]string{
		0:          "",
		512:        "512 B",
		2048:       "2 KB",
		522928832:  "499 MB",
		4700000000: "4.4 GB",
	}
	for in, want := range cases {
		if got := sizeOf(in); got != want {
			t.Errorf("sizeOf(%d) = %q, want %q", in, got, want)
		}
	}
}

// TestMadeOf_whatAModelIsBuiltOf checks the line under a model the machine
// holds: what it is made of, and nothing when the machine said nothing.
func TestMadeOf_whatAModelIsBuiltOf(t *testing.T) {
	got := madeOf(ai.OllamaModel{
		Family: "qwen3", ParameterSize: "0.6B", Quantization: "Q8_0",
	})
	if got != "qwen3 · 0.6B · Q8_0" {
		t.Fatalf("made of = %q, want all three", got)
	}
	if made := madeOf(ai.OllamaModel{Name: "x"}); made != "" {
		t.Fatalf("made of = %q, want nothing said", made)
	}
}
