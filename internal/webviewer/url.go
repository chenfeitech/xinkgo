// Package webviewer holds the rules of the Web viewer's address bar. It is
// pure: no windows, no WebView, no I/O. The app owns the native page and
// decides what to do with the URLs this package says are usable.
package webviewer

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"unicode"
)

// Normalize turns what was typed into an address. Text that looks like a
// host becomes https; anything else is a search. A scheme other than
// http/https is refused, and loopback/private hosts are refused unless the
// user asked for them.
func Normalize(raw, searchEngine string, allowLocal bool) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("empty address")
	}
	if strings.Contains(raw, "://") {
		return checkURL(raw, allowLocal)
	}
	if looksLikeHost(raw) {
		return checkURL("https://"+raw, allowLocal)
	}
	return SearchURL(searchEngine, raw)
}

// looksLikeHost reports whether text with no scheme is probably a host:
// no spaces, and at least one dot or a colon followed by a port.
func looksLikeHost(raw string) bool {
	if strings.ContainsAny(raw, " \t\n") {
		return false
	}
	host := raw
	if i := strings.IndexAny(raw, "/?#"); i >= 0 {
		host = raw[:i]
	}
	return strings.Contains(host, ".") || strings.Contains(host, ":")
}

// SearchURL fills a search engine template. The query is escaped, so a
// search for "a & b" cannot turn into two query parameters.
func SearchURL(template, query string) (string, error) {
	if !strings.Contains(template, "{query}") {
		return "", errors.New("the search engine needs {query}")
	}
	address := strings.ReplaceAll(template, "{query}",
		url.QueryEscape(query))
	parsed, err := url.Parse(address)
	if err != nil || parsed.Scheme != "https" && parsed.Scheme != "http" ||
		parsed.Host == "" {
		return "", fmt.Errorf("the search engine is not a usable URL")
	}
	return address, nil
}

// IsAllowed reports whether an address may be loaded. It is used again on
// navigation events, where a page tries to send the viewer somewhere else.
func IsAllowed(raw string, allowLocal bool) bool {
	_, err := checkURL(raw, allowLocal)
	return err == nil
}

func checkURL(raw string, allowLocal bool) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("%q is not an http(s) address", parsed.Scheme)
	}
	if parsed.Host == "" {
		return "", errors.New("the address has no host")
	}
	if !allowLocal && localHost(parsed.Hostname()) {
		return "", errors.New("local addresses are turned off")
	}
	return parsed.String(), nil
}

// localHost reports loopback, link-local and private addresses. A name
// that is not an IP (an intranet DNS name) cannot be checked here; the
// residual risk is documented rather than pretended away.
func localHost(host string) bool {
	if strings.EqualFold(host, "localhost") ||
		strings.HasSuffix(strings.ToLower(host), ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified()
}

// CleanTitle turns a page title into a usable note name: no path
// separators, no control characters, and a bound on its length.
func CleanTitle(title string) string {
	title = strings.TrimSpace(title)
	var b strings.Builder
	written := 0
	for _, r := range title {
		if unicode.IsControl(r) || strings.ContainsRune(`/\?%*:|"<>`, r) {
			continue
		}
		b.WriteRune(r)
		written++
		if written >= 80 {
			break
		}
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "page"
	}
	return out
}
