package app

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestShim_requestUrlBranchesAreHandled guards a runtime-only defect.
// requestUrl attaches .text/.json/.arrayBuffer promise branches; when those
// are plain promise.then(...) calls, a denied network request leaves three
// unhandled rejections behind and the host marks the plugin as failed even
// if the caller caught the main promise. The shim is JS, so this checks the
// shape of the source rather than executing it.
func TestShim_requestUrlBranchesAreHandled(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller could not locate this test file")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..",
		"resources", "obsidian-shim", "shim.js"))
	if err != nil {
		t.Fatalf("shim.js: %v", err)
	}
	src := string(data)
	start := strings.Index(src, "function requestUrl(")
	if start < 0 {
		t.Fatal("requestUrl was not found in shim.js")
	}
	tail := strings.Index(src[start:], "return promise;")
	if tail < 0 {
		t.Fatal("requestUrl has no return promise")
	}
	section := src[start : start+tail+len("return promise;")]
	for _, name := range []string{"arrayBuffer", "json", "text"} {
		want := "promise." + name + " = branch(\"" + name + "\");"
		if !strings.Contains(section, want) {
			t.Errorf("requestUrl .%s branch is not wrapped: want %s",
				name, want)
		}
	}
	if !strings.Contains(section, "one.catch(function () {});") {
		t.Error("requestUrl branches have no rejection handler")
	}
}

// TestShim_exposesBothModuleNames guards the alias: official plugins use
// require("xinkgo") while external ones keep require("obsidian"), and both
// resolve to the same exports object.
func TestShim_exposesBothModuleNames(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller could not locate this test file")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..",
		"resources", "obsidian-shim", "shim.js"))
	if err != nil {
		t.Fatalf("shim.js: %v", err)
	}
	src := string(data)
	if !strings.Contains(src,
		`name === "obsidian" || name === "xinkgo"`) {
		t.Error("require does not accept both host module names")
	}
	if !strings.Contains(src,
		`window.xinkgo = window.require("xinkgo")`) {
		t.Error("window.xinkgo is not exposed")
	}
}
