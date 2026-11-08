"use strict";

// Xinkgo plugin demo: every capability the host fully supports.
//
// Shape the host expects: a CommonJS bundle, one class extending
// obsidian.Plugin, and module.exports = that class. No ESM syntax, no
// relative modules, no dynamic module names.
//
// This bundle only calls APIs with a real Xinkgo implementation: native
// commands / notices / saved data / ribbon / status bar, the Vault and
// Editor bridges, workspace.openLinkText, requestUrl, app events, and the
// plugin-owned ItemView surface. APIs that the host merely records or fakes
// are deliberately absent; the reasons are in examples/plugins/README.md.
//
// The host reports C3 for this bundle because registerView is a DOM-level
// feature. That surface is real: the host lists the view in the plugin view
// drawer, builds the ItemView and mounts containerEl in this page.

var obsidian = require("obsidian");

var PREFIX = "Xinkgo Demo: Full";

function log(what, detail) {
  console.log(PREFIX + " | " + what + " | " + String(detail || ""));
}

function say(text) {
  new obsidian.Notice(text, 4000);
}

function fail(what, err) {
  var text = String((err && err.message) || err || "unknown error");
  console.error(PREFIX + " " + what + " failed: " + text);
  new obsidian.Notice(what + " failed: " + text, 6000);
}

function activeEditor(app) {
  var view = app.workspace.getActiveViewOfType(obsidian.MarkdownView);
  return view ? view.editor : null;
}

// ---------------------------------------------------------------------------
// A plugin-owned ItemView. The host creates a WorkspaceLeaf stub, calls the
// creator and appends view.containerEl to the plugin page's root element.
// This is a real Xinkgo surface, not a stand-in.
// ---------------------------------------------------------------------------
function DemoView(leaf, plugin) {
  obsidian.ItemView.call(this, leaf);
  this.plugin = plugin;
  this.icon = "layers";
  this.ticks = 0;
}
DemoView.prototype = Object.create(obsidian.ItemView.prototype);
DemoView.prototype.constructor = DemoView;
DemoView.prototype.getViewType = function () { return "demo-full-view"; };
DemoView.prototype.getDisplayText = function () { return "Xinkgo Demo: Full view"; };
DemoView.prototype.getIcon = function () { return "layers"; };
DemoView.prototype.getState = function () { return { ticks: this.ticks }; };
DemoView.prototype.setState = function (state) {
  if (state && typeof state.ticks === "number") this.ticks = state.ticks;
};
DemoView.prototype.onOpen = function () {
  var self = this;
  var root = this.contentEl;
  this.containerEl.addClass("xinkgo-demo-full-view");
  root.empty();
  root.createEl("h2", { text: "Xinkgo Demo: Full" });
  root.createEl("p", {
    text: "A real plugin-owned view. Picking demo-full-view in the plugin " +
      "view drawer made the host append this ItemView container."
  });
  var badge = root.createEl("img", { cls: "xinkgo-demo-full-badge" });
  badge.setAttr("src", "/plugin/assets/badge.svg");
  badge.setAttr("alt", "Demo badge");
  var button = root.createEl("button", {
    cls: "xinkgo-demo-full-button", text: "Add a row"
  });
  button.addEventListener("click", function () {
    self.ticks += 1;
    root.createDiv({ cls: "xinkgo-demo-full-row", text: "Row " + self.ticks });
  });
  log("ItemView.onOpen", "containerEl and contentEl are real DOM nodes here");
};
DemoView.prototype.onClose = function () {
  log("ItemView.onClose", "the host detaches the container");
};
DemoView.prototype.onResize = function () {
  log("ItemView.onResize", "called when the page is shown");
};

// ---------------------------------------------------------------------------
// The plugin itself.
// ---------------------------------------------------------------------------
function DemoFull(app, manifest) {
  obsidian.Plugin.call(this, app, manifest);
  this.state = { runs: 0, lastPath: "" };
  this.status = null;
  this.temporaryCommand = false;
}
DemoFull.prototype = Object.create(obsidian.Plugin.prototype);
DemoFull.prototype.constructor = DemoFull;

DemoFull.prototype.onload = function () {
  var self = this;
  return this.loadData().then(function (saved) {
    if (saved && typeof saved === "object") {
      if (typeof saved.runs === "number") self.state.runs = saved.runs;
      if (typeof saved.lastPath === "string") self.state.lastPath = saved.lastPath;
    }
    self.status = self.addStatusBarItem();
    self.refreshStatus();
    self.addRibbonIcon("layers", "Demo Full: Print the tour", function () {
      self.tour();
    });
    self.addRibbonIcon("puzzle", "Demo Full: Say hello", function () {
      self.hello();
    });

    self.registerCommands();
    self.registerViewSurface();
    self.registerAppEvents();

    new obsidian.Notice(
      "Xinkgo Demo: Full loaded (real capabilities only)", 5000);
    log("level", "C3: native adapters plus a real plugin-owned view");
  }).catch(function (err) {
    fail("onload", err);
  });
};

DemoFull.prototype.onunload = function () {
  log("onunload", "Component cleanups run automatically");
};

DemoFull.prototype.refreshStatus = function () {
  if (this.status) {
    this.status.setText("Demo Full: " + this.state.runs + " runs");
  }
};

DemoFull.prototype.record = function () {
  this.state.runs += 1;
  this.saveData(this.state).catch(function (err) { fail("saveData", err); });
  this.refreshStatus();
};

DemoFull.prototype.tour = function () {
  var caps = [
    "data", "commands", "notice", "vault.read", "vault.write",
    "editor.read", "editor.write", "network", "events.app", "ui", "dom"
  ];
  log("capabilities", caps.join(", "));
  log("level", "every capability here has a real Xinkgo implementation");
  this.record();
  say("Full demo covers " + caps.length +
    " supported capabilities; details are in the plugin log");
};

DemoFull.prototype.hello = function () {
  this.record();
  say("Hello from Xinkgo Demo: Full (run #" + this.state.runs + ")");
};

DemoFull.prototype.registerCommands = function () {
  var self = this;

  self.addCommand({
    id: "tour", name: "Print the capability tour",
    callback: function () { self.tour(); }
  });
  self.addCommand({
    id: "hello", name: "Say hello and persist a counter",
    callback: function () { self.hello(); }
  });
  self.addCommand({
    id: "read-note", name: "Read the active note (vault.read)",
    callback: function () { self.readNote(); }
  });
  self.addCommand({
    id: "insert-summary", name: "Insert a line at the caret (editor.write)",
    editorCallback: function (editor) {
      editor.replaceSelection("\n\n[Demo Full] " +
        obsidian.moment().format("YYYY-MM-DD HH:mm:ss") + "\n");
      self.record();
      say("Inserted a line at the caret");
    }
  });
  self.addCommand({
    id: "replace-buffer", name: "Replace the whole buffer (editor.write)",
    editorCallback: function (editor) {
      editor.setValue("# Demo Full\n\nThe buffer was replaced by an " +
        "Obsidian-compatible plugin.\n");
      self.record();
      say("Buffer replaced");
    }
  });
  self.addCommand({
    id: "selection-stats", name: "Show selection stats (needs a selection)",
    editorCheckCallback: function (checking, editor) {
      if (!editor.somethingSelected()) return false;
      if (checking) return true;
      self.record();
      say("Selection: " + editor.getSelection().length + " characters");
      return true;
    }
  });
  self.addCommand({
    id: "vault-lifecycle", name: "Run a Vault lifecycle (needs a note)",
    checkCallback: function (checking) {
      if (!self.app.workspace.getActiveFile()) return false;
      if (checking) return true;
      self.vaultLifecycle();
      return true;
    }
  });
  self.addCommand({
    id: "open-note-by-link", name: "Open the active note by link",
    callback: function () { self.openNoteByLink(); }
  });
  self.addCommand({
    id: "network-request", name: "Fetch example.com (network)",
    callback: function () { self.requestSample(); }
  });
  self.addCommand({
    id: "toggle-command", name: "Toggle a temporary command",
    callback: function () { self.toggleCommand(); }
  });
  self.addCommand({
    id: "open-view", name: "Where is the demo view?",
    callback: function () { self.openViewHint(); }
  });
};

DemoFull.prototype.toggleCommand = function () {
  var self = this;
  if (this.temporaryCommand) {
    this.removeCommand("temporary");
    this.temporaryCommand = false;
    say("Removed the temporary command");
    return;
  }
  this.addCommand({
    id: "temporary", name: "Temporary command",
    callback: function () { self.hello(); }
  });
  this.temporaryCommand = true;
  say("Added a temporary command");
};

DemoFull.prototype.openViewHint = function () {
  log("registerView", "demo-full-view is listed in the plugin view drawer");
  this.record();
  say("Pick demo-full-view in the plugin view drawer on the right");
};

DemoFull.prototype.readNote = function () {
  var self = this;
  var file = this.app.workspace.getActiveFile();
  if (!file) {
    say("Open a note first");
    return;
  }
  var editor = activeEditor(this.app);
  if (editor) {
    log("editor.getValue", String(editor.getValue()).length + " characters");
    log("editor.getSelection",
      JSON.stringify(editor.getSelection().slice(0, 40)));
    log("editor.getCursor", JSON.stringify(editor.getCursor()));
    log("editor.getRange", JSON.stringify(editor.getRange(
      { line: 0, ch: 0 }, { line: 0, ch: 20 })));
  }
  return this.app.vault.read(file).then(function (text) {
    log("vault.read", String(text).length + " characters");
    return self.app.vault.cachedRead(file);
  }).then(function (text) {
    log("vault.cachedRead", String(text).length + " characters");
    return self.app.vault.adapter.read(file.path);
  }).then(function (text) {
    log("vault.adapter.read", String(text).length + " characters");
    self.record();
    say("Read the active note through three adapters");
  }).catch(function (err) {
    fail("vault.read", err);
  });
};

DemoFull.prototype.vaultLifecycle = function () {
  var self = this;
  var vault = this.app.vault;
  var dir = "demo-full";
  var first = dir + "/lifecycle.md";
  var renamed = dir + "/lifecycle-renamed.md";
  // Every write goes through the vault.write bridge; a missing grant
  // rejects the promise instead of throwing inside onload.
  return vault.createFolder(dir)
    .catch(function () { return null; })
    .then(function () { return vault.create(first, "# Lifecycle\n\n"); })
    .then(function () { return vault.modify(first, "# Lifecycle\n\nmodified\n"); })
    .then(function () { return vault.cachedRead(first); })
    .then(function (text) {
      log("vault.cachedRead", String(text).length + " characters");
      return vault.append(first, "appended\n");
    })
    .then(function () { return vault.rename(first, renamed); })
    .then(function () { return vault.delete(renamed); })
    .then(function () {
      self.record();
      say("Vault lifecycle: created, modified, appended, renamed, deleted");
    })
    .catch(function (err) { fail("Vault lifecycle", err); });
};

DemoFull.prototype.openNoteByLink = function () {
  var self = this;
  var file = this.app.workspace.getActiveFile();
  if (!file) {
    say("Open a note first");
    return;
  }
  log("workspace.openLinkText", file.path);
  this.app.workspace.openLinkText(file.path).then(function () {
    self.record();
    say("Opened " + file.path + " through workspace.openLinkText");
  }).catch(function (err) {
    fail("workspace.openLinkText", err);
  });
};

DemoFull.prototype.requestSample = function () {
  var self = this;
  // Only an explicit command reaches the network; onload never does.
  say("Requesting https://example.com (needs the network grant)");
  obsidian.requestUrl({ url: "https://example.com", method: "GET" })
    .then(function (response) {
      return Promise.resolve(response.text).then(function (body) {
        log("requestUrl", "status " + response.status + ", " +
          String(body).length + " characters");
        self.record();
        say("requestUrl returned status " + response.status);
      });
    })
    .catch(function (err) { fail("requestUrl", err); });
};

DemoFull.prototype.registerViewSurface = function () {
  var self = this;
  // registerView is a real Xinkgo surface: the host lists one visible view
  // in the drawer on the right of the main window, then builds the ItemView.
  this.registerView("demo-full-view", function (leaf) {
    return new DemoView(leaf, self);
  });
  log("registerView", "demo-full-view registered with the host");
};

DemoFull.prototype.registerAppEvents = function () {
  var self = this;
  // Host events are signals: they carry no file object, so read the active
  // note yourself when you need its path.
  this.registerEvent(this.app.workspace.on("file-open", function () {
    var active = self.app.workspace.getActiveFile();
    var path = active ? active.path : "";
    self.state.lastPath = path;
    self.saveData(self.state).catch(function (err) { fail("saveData", err); });
    log("file-open", path || "(no active note)");
  }));
  this.registerEvent(this.app.vault.on("modify", function () {
    log("vault modify", "a vault file changed");
  }));
  this.registerEvent(this.app.workspace.on("editor-change", function () {
    log("editor-change", "the editor buffer changed");
  }));
  this.registerDomEvent(window.activeDocument, "click", function () {
    log("registerDomEvent", "click on the plugin page");
  });
  var ticks = 0;
  this.registerInterval(window.setInterval(function () {
    ticks += 1;
    if (ticks <= 3) log("registerInterval", "tick " + ticks);
  }, 60000));
  log("events.app", "file-open / vault.modify / editor-change / dom / interval");
};

module.exports = DemoFull;
