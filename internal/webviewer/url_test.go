package webviewer

import (
	"strings"
	"testing"
)

func TestNormalize_hostsAndSearch(t *testing.T) {
	engine := "https://duckduckgo.com/?q={query}"
	for _, one := range []struct {
		in, want string
	}{
		{"example.com", "https://example.com"},
		{"https://example.com/a?b=1", "https://example.com/a?b=1"},
		{"example.com:8080/x", "https://example.com:8080/x"},
		{"hello world", "https://duckduckgo.com/?q=hello+world"},
		{"a & b", "https://duckduckgo.com/?q=a+%26+b"},
	} {
		got, err := Normalize(one.in, engine, false)
		if err != nil {
			t.Errorf("Normalize(%q): %v", one.in, err)
			continue
		}
		if got != one.want {
			t.Errorf("Normalize(%q) = %q, want %q", one.in, got, one.want)
		}
	}
}

func TestNormalize_refusesBadSchemesAndLocalHosts(t *testing.T) {
	engine := "https://duckduckgo.com/?q={query}"
	for _, one := range []string{
		"file:///etc/passwd", "javascript:alert(1)", "xinkgo-plugin://x/",
		"http://localhost:3000", "http://127.0.0.1",
		"http://10.0.0.1", "http://192.168.1.1", "http://[::1]",
		"https://169.254.1.1", "  ",
	} {
		if got, err := Normalize(one, engine, false); err == nil {
			t.Errorf("Normalize(%q) = %q, should be refused", one, got)
		}
	}
	if got, err := Normalize("http://localhost:3000", engine, true); err != nil ||
		got != "http://localhost:3000" {
		t.Errorf("allowed local address was refused: %q, %v", got, err)
	}
}

func TestSearchURL_needsPlaceholder(t *testing.T) {
	if _, err := SearchURL("https://example.com/search", "x"); err == nil {
		t.Error("a template without {query} was accepted")
	}
}

func TestCleanTitle(t *testing.T) {
	if got := CleanTitle("  A/B:C?  "); got != "ABC" {
		t.Errorf("CleanTitle = %q", got)
	}
	if got := CleanTitle("   "); got != "page" {
		t.Errorf("empty title = %q", got)
	}
	if got := CleanTitle(strings.Repeat("长", 100)); len([]rune(got)) != 80 {
		t.Errorf("long title was not bounded: %d runes", len([]rune(got)))
	}
}
