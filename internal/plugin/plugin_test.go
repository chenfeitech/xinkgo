package plugin

import (
	"strings"
	"testing"
)

const validManifest = `{
  "id":"demo-addon","name":"Demo","version":"1.2.3",
  "minAppVersion":"0.9.12","description":"A demo","author":"Someone",
  "isDesktopOnly":false
}`

func TestParseManifest_acceptsUnknownFieldsAndMissingMinimum(t *testing.T) {
	m, err := ParseManifest([]byte(`{
  "id":"demo-addon","name":"Demo","version":"1.0.0",
  "description":"A demo","author":"Someone","fundingUrl":"https://example.com",
  "isDesktopOnly":false
}`))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if m.MinAppVersion != "0.0.0" {
		t.Errorf("minAppVersion = %q, want 0.0.0", m.MinAppVersion)
	}
	if len(m.FundingURL) == 0 {
		t.Error("fundingUrl was dropped")
	}
}

func TestValidateManifest_rejectsBadIDAndFolderMismatch(t *testing.T) {
	m, err := ParseManifest([]byte(validManifest))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateManifest(m, "other-folder"); err == nil {
		t.Error("a folder that does not match the id was accepted")
	}
	m.ID = "Bad-Plugin"
	if err := ValidateManifest(m, ""); err == nil {
		t.Error("an uppercase id was accepted")
	}
	// Legacy community plugins are allowed to have "obsidian" in the ID.
	m.ID = "obsidian-legacy"
	if err := ValidateManifest(m, ""); err != nil {
		t.Errorf("a published-style id was refused: %v", err)
	}
}

func TestCompareVersions_comparesNumericCore(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{{"1.0.0", "0.9.12", 1}, {"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.5.0", -1}, {"1.2.3-beta", "1.2.3", 0}}
	for _, one := range cases {
		got, err := CompareVersions(one.a, one.b)
		if err != nil || got != one.want {
			t.Errorf("CompareVersions(%q,%q) = %d,%v, want %d",
				one.a, one.b, got, err, one.want)
		}
	}
}

func TestScanSource_acceptsCoreCommands(t *testing.T) {
	src := []byte(`"use strict";var o=require("obsidian");
module.exports=class extends o.Plugin{async onload(){this.addCommand({id:"x",name:"X",callback:()=>new o.Notice("hi")});}}`)
	r := ScanSource(src)
	if !r.Supported || r.Level != LevelCore {
		t.Fatalf("supported=%v level=%v issues=%v", r.Supported,
			r.Level.Code(), r.Issues)
	}
	if !containsCapability(r.Capabilities, CapabilityCommands) ||
		!containsCapability(r.Capabilities, CapabilityNotice) {
		t.Errorf("capabilities = %v", r.Capabilities)
	}
}

func TestScanSource_refusesNodeCM6AndDOM(t *testing.T) {
	cases := map[string]string{
		"node":    `var fs=require("fs");module.exports=class extends require("obsidian").Plugin{}`,
		"cm6":     `var cm=require("@codemirror/state");module.exports=class extends require("obsidian").Plugin{}`,
		"view":    `module.exports=class extends require("obsidian").Plugin{onload(){this.registerView("x",()=>0)}}`,
		"modal":   `var o=require("obsidian");module.exports=class extends o.Plugin{onload(){new o.Modal()}}`,
		"dynamic": `var n="fs";require(n);module.exports=class extends require("obsidian").Plugin{}`,
		"esm":     `import {Plugin} from "obsidian"; export default class extends Plugin{}`,
	}
	for name, src := range cases {
		r := ScanSource([]byte(src))
		if r.Supported || len(r.Issues) == 0 {
			t.Errorf("%s: supported=%v issues=%v", name, r.Supported,
				r.Issues)
		}
	}
}

func TestScanPackage_refusesANewerAPIProfile(t *testing.T) {
	m, err := ParseManifest([]byte(strings.Replace(validManifest,
		`"minAppVersion":"0.9.12"`, `"minAppVersion":"1.5.0"`, 1)))
	if err != nil {
		t.Fatal(err)
	}
	r := ScanPackage(m, []byte(`module.exports=class extends require("obsidian").Plugin{}`),
		"/tmp/demo-addon", "user", false)
	if r.Supported {
		t.Errorf("a plugin asking for Obsidian 1.5 was accepted: %+v", r)
	}
}

func TestFilterCatalog_supportedOnlyHidesUncheckedPlugins(t *testing.T) {
	entries, err := ParseCatalog([]byte(`[
  {"id":"one","name":"One","author":"A","description":"D","repo":"a/one"},
  {"id":"two","name":"Two","author":"B","description":"D","repo":"b/two"}
]`))
	if err != nil {
		t.Fatal(err)
	}
	compatible := []CompatibleEntry{{ID: "one", Repo: "a/one", Level: "C1"}}
	got := FilterCatalog(entries, compatible, "", true, 0)
	if len(got) != 1 || got[0].ID != "one" || !got[0].Compatible {
		t.Fatalf("supported only = %+v", got)
	}
	got = FilterCatalog(entries, compatible, "Two", false, 0)
	if len(got) != 1 || got[0].ID != "two" || got[0].Compatible {
		t.Fatalf("query = %+v", got)
	}
}

func containsCapability(in []Capability, want Capability) bool {
	for _, one := range in {
		if one == want {
			return true
		}
	}
	return false
}
