package locale

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/helightxu/xinkgo/internal/config"
)

// TestCatalogs_theSameKeys checks that every language says the same
// messages as the source language: a message the translation leaves out
// falls back to English, which is a thing the reader notices and the
// build does not.
func TestCatalogs_theSameKeys(t *testing.T) {
	source := keysOf(t, "en.json")
	for _, name := range []string{"zh-Hans.json"} {
		got := keysOf(t, name)
		for key := range source {
			if !got[key] {
				t.Errorf("%s has no %q", name, key)
			}
		}
		for key := range got {
			if !source[key] {
				t.Errorf("%s has %q, which en.json does not", name, key)
			}
		}
	}
}

// keysOf is the messages of a catalog, its metadata left out.
func keysOf(t *testing.T, name string) map[string]bool {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(".", name))
	if err != nil {
		t.Fatal(err)
	}
	var file map[string]json.RawMessage
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	out := map[string]bool{}
	for key := range file {
		if len(key) > 0 && key[0] == '@' {
			continue // the metadata of a message, and of the file
		}
		out[key] = true
	}
	return out
}

// TestIn_theLanguageAskedFor checks that a message is said in the
// language it is asked for, and not in the one the app shows: the
// settings name a language, and the app says it to MyGo, which is what
// makes everything else follow it.
func TestIn_theLanguageAskedFor(t *testing.T) {
	if got := In("zh-Hans").MenuSettings(); got != "设置…" {
		t.Errorf("in Chinese the settings are called %q", got)
	}
	if got := MenuSettings(); got != "Settings…" {
		t.Errorf("with no language chosen the settings are called %q", got)
	}
}

// TestEveryLanguageOfTheConfigurationIsInTheCatalogs checks that the
// languages the settings offer are languages the app is written in: a
// language it has no messages for is English under another name.
func TestEveryLanguageOfTheConfigurationIsInTheCatalogs(t *testing.T) {
	have := Catalog.Languages()
	for _, lang := range []string{
		config.LanguageEnglish, config.LanguageChinese,
	} {
		if !slices.Contains(have, lang) {
			t.Errorf("the settings offer %q, which the catalogs do not "+
				"have (%v)", lang, have)
		}
		if got := In(lang).MenuSettings(); got == "" ||
			got == MenuSettings() && lang != config.LanguageEnglish {
			t.Errorf("%q says the settings as %q, want its own words",
				lang, got)
		}
	}
}
