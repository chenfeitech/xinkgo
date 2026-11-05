package plugin

import "testing"

func TestCorePlugins_haveUniqueStableIDs(t *testing.T) {
	seen := map[string]bool{}
	for _, one := range CorePlugins {
		if one.ID == "" || seen[one.ID] {
			t.Fatalf("empty or duplicate core id: %q", one.ID)
		}
		seen[one.ID] = true
	}
	if len(CoreIDs()) != len(CorePlugins) {
		t.Fatal("CoreIDs does not cover the registry")
	}
}

func TestCorePlugins_defaultsCoverEveryID(t *testing.T) {
	def := DefaultCore()
	for _, one := range CorePlugins {
		if !one.Default {
			continue
		}
		if !def[one.ID] {
			t.Fatalf("default core is missing %q", one.ID)
		}
	}
	if len(def) > len(CorePlugins) {
		t.Fatal("defaults name an id outside the registry")
	}
}

func TestIsCore(t *testing.T) {
	if !IsCore(CorePlugins[0].ID) || IsCore("some-community-plugin") {
		t.Fatal("IsCore did not recognise a built-in id")
	}
}
