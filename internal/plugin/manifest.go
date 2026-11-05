package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Manifest is the metadata Obsidian puts in a plugin's manifest.json.
// Unknown fields are left alone; Xinkgo's own hints are optional.
type Manifest struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Version       string          `json:"version"`
	MinAppVersion string          `json:"minAppVersion"`
	Description   string          `json:"description"`
	Author        string          `json:"author"`
	AuthorURL     string          `json:"authorUrl,omitempty"`
	FundingURL    json.RawMessage `json:"fundingUrl,omitempty"`
	IsDesktopOnly bool            `json:"isDesktopOnly,omitempty"`
	Dir           string          `json:"dir,omitempty"`
	Xinkgo        *ManifestXinkgo `json:"xinkgo,omitempty"`
}

// ManifestXinkgo holds hints Obsidian ignores and Xinkgo may use.
type ManifestXinkgo struct {
	Capabilities []string `json:"capabilities,omitempty"`
}

var idPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$`)

// ParseManifest reads a plugin manifest without touching the filesystem.
func ParseManifest(data []byte) (Manifest, error) {
	var m Manifest
	if len(data) == 0 {
		return m, errors.New("empty manifest")
	}
	dec := json.NewDecoder(strings.NewReader(string(data)))
	if err := dec.Decode(&m); err != nil {
		return m, fmt.Errorf("parse manifest: %w", err)
	}
	// A few published releases leave minAppVersion as null. Treat that as
	// "no minimum" instead of refusing an otherwise valid plugin.
	if strings.TrimSpace(m.MinAppVersion) == "" {
		m.MinAppVersion = "0.0.0"
	}
	if err := ValidateManifest(m, ""); err != nil {
		return m, err
	}
	return m, nil
}

// ValidateManifest checks the fields a plugin host must rely on. folder,
// when given, must match the ID, as Obsidian's local development rules ask.
func ValidateManifest(m Manifest, folder string) error {
	for key, value := range map[string]string{
		"id": m.ID, "name": m.Name, "version": m.Version,
		"description": m.Description, "author": m.Author,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("manifest: %s is empty", key)
		}
	}
	// The published community catalog predates Obsidian's current naming
	// rules for new submissions, so a compatibility host must accept IDs
	// such as "obsidian-underline" and "hotkeysplus-obsidian".
	if !idPattern.MatchString(m.ID) {
		return fmt.Errorf("manifest: invalid id %q", m.ID)
	}
	if folder != "" && folder != m.ID {
		return fmt.Errorf("manifest: id %q does not match folder %q",
			m.ID, folder)
	}
	if !versionPattern.MatchString(m.Version) {
		return fmt.Errorf("manifest: invalid version %q", m.Version)
	}
	if m.MinAppVersion != "" && !versionPattern.MatchString(m.MinAppVersion) {
		return fmt.Errorf("manifest: invalid minAppVersion %q",
			m.MinAppVersion)
	}
	return nil
}

// CompareVersions compares two semantic version strings by their numeric
// core. Pre-release suffixes are ignored: the compatibility profile only
// needs to know whether a plugin asks for a newer core API.
func CompareVersions(a, b string) (int, error) {
	parse := func(v string) ([]int, error) {
		v = strings.SplitN(v, "-", 2)[0]
		v = strings.SplitN(v, "+", 2)[0]
		parts := strings.Split(v, ".")
		if len(parts) != 3 {
			return nil, fmt.Errorf("bad version %q", v)
		}
		out := make([]int, 3)
		for i, part := range parts {
			n, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("bad version %q", v)
			}
			out[i] = n
		}
		return out, nil
	}
	left, err := parse(a)
	if err != nil {
		return 0, err
	}
	right, err := parse(b)
	if err != nil {
		return 0, err
	}
	for i := range left {
		if left[i] < right[i] {
			return -1, nil
		}
		if left[i] > right[i] {
			return 1, nil
		}
	}
	return 0, nil
}

// RequireAPIVersion reports whether this host's profile is new enough for
// the Obsidian version a plugin asks for.
func RequireAPIVersion(version string) bool {
	cmp, err := CompareVersions(APIProfile, version)
	return err == nil && cmp >= 0
}
