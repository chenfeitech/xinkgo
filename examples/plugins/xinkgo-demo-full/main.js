"use strict";

// Xinkgo plugin demo: a tour of every capability the host exposes.
//
// Shape the host expects: a CommonJS bundle, one class extending
// obsidian.Plugin, and module.exports = that class. No ESM syntax, no
// relative modules, no dynamic module names.
//
// This demo deliberately asks for the whole API surface, so the host
// reports it as C4 / degraded. The honest boundary:
//   - commands, notices, data, Vault, Editor, metadata, network, events,
//     ribbon and status items go through real host adapters;
//   - ItemView, Modal, Menu, Setting and MarkdownRenderer draw into this
//     plugin's own WebView page, never into Xinkgo's native UI;
//   - CodeMirror 6 extensions, protocol handlers and Node built-ins are
//     recorded or replaced by inert stubs.
// Every permission the demo needs is listed in manifest.json under
// "xinkgo". The user still grants each one in the plugin manager.

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
    text: "This is the plugin's own DOM surface. Picking the view in the " +
      "plugin view drawer made the host append this ItemView container."
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
// A settings-like tab. addSettingTab calls display() right away, but the
// page is this plugin's WebView, not Xinkgo's native settings window.
// ---------------------------------------------------------------------------
function DemoSettingTab(app, plugin) {
  obsidian.PluginSettingTab.call(this, app, plugin);
}
DemoSettingTab.prototype = Object.create(obsidian.PluginSettingTab.prototype);
DemoSettingTab.prototype.constructor = DemoSettingTab;
DemoSettingTab.prototype.display = function () {
  this.containerEl.empty();
  this.containerEl.addClass("xinkgo-demo-full-settings");
  this.containerEl.createEl("h2", { text: "Demo Full settings (stub)" });
  this.containerEl.createEl("p", {
    text: "addSettingTab draws into the plugin WebView page. Real settings " +
      "belong in loadData / saveData."
  });
  new obsidian.Setting(this.containerEl)
    .setName("Example toggle")
    .setDesc("Controls are inert stubs on this page.")
    .addToggle(function (toggle) {
      toggle.setValue(false).onChange(function (value) {
        log("Setting.addToggle", "stub value " + String(value));
      });
    });
  new obsidian.Setting(this.containerEl)
    .setName("Example button")
    .addButton(function (button) {
      button.setIcon("layers").setCta();
      log("Setting.addButton", "constructed an inert button stub");
    });
};

// ---------------------------------------------------------------------------
// The plugin itself.
// ---------------------------------------------------------------------------
function DemoFull(app, manifest) {
  obsidian.Plugin.call(this, app, manifest);
  this.state = { runs: 0, lastPath: "" };
  this.status = null;
  this.temporaryCommand = false;
  this.counts = { extensions: 0, suggests: 0, processors: 0, protocol: 0 };
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
    self.registerSurfaces();
    self.registerExtensionsAndStubs();
    self.registerAppEvents();

    new obsidian.Notice("Xinkgo Demo: Full loaded (C4, degraded)", 5000);
    log("level", "C4 / degraded: DOM, CM6 and Node modules are shims");
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
    "editor.read", "editor.write", "metadata", "network",
    "events.app", "ui", "dom", "editor.extension", "compat.stub"
  ];
  log("capabilities", caps.join(", "));
  log("level", "C4 / degraded: DOM, CM6 and Node modules are shims");
  this.record();
  say("Full demo covers all 14 capabilities; details are in the plugin log");
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
    id: "metadata-and-links", name: "Inspect metadata and open the note",
    callback: function () { self.showMetadata(); }
  });
  self.addCommand({
    id: "network-request", name: "Fetch example.com (network)",
    callback: function () { self.requestSample(); }
  });
  self.addCommand({
    id: "dom-samples", name: "Build DOM samples in the plugin page",
    callback: function () { self.showDomSamples(); }
  });
  self.addCommand({
    id: "editor-extensions", name: "Inspect registered editor extensions",
    callback: function () { self.showEditorExtensions(); }
  });
  self.addCommand({
    id: "compat-stubs", name: "Exercise compatibility stubs",
    callback: function () { self.exerciseStubs(); }
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

DemoFull.prototype.showMetadata = function () {
  var self = this;
  var file = this.app.workspace.getActiveFile();
  if (!file) {
    say("Open a note first");
    return;
  }
  var cache = this.app.metadataCache.getFileCache(file);
  log("MetadataCache.getFileCache", JSON.stringify(cache || {}));
  var dest = this.app.metadataCache.getFirstLinkpathDest(file.path);
  log("getFirstLinkpathDest", dest ? dest.path : "no match");
  var link = this.app.fileManager.generateMarkdownLink(
    file, "", "", "Demo link");
  log("FileManager.generateMarkdownLink", link);
  this.app.workspace.openLinkText(file.path).then(function () {
    self.record();
    say("openLinkText returned (needs vault.read)");
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

DemoFull.prototype.showDomSamples = function () {
  var self = this;
  var modal = new obsidian.Modal(this.app);
  modal.containerEl.addClass("xinkgo-demo-full-modal");
  modal.titleEl.setText("Demo Full modal (plugin page only)");
  modal.contentEl.createEl("p", {
    text: "This modal lives in the plugin WebView, not in Xinkgo's window."
  });
  modal.open();
  window.setTimeout(function () { modal.close(); }, 3000);

  var menu = new obsidian.Menu(this.app);
  menu.addItem(function (item) {
    item.setTitle("Demo MenuItem").setIcon("layers").onClick(function () {
      say("Menu item chosen (stub)");
    });
  });
  menu.addSeparator();
  var standalone = new obsidian.MenuItem(menu);
  standalone.setTitle("Standalone MenuItem");
  log("new Menu", menu.items.length + " items; native menus are untouched");

  var suggest = new obsidian.SuggestModal(this.app);
  suggest.setPlaceholder("Type something");
  var fuzzy = new obsidian.FuzzySuggestModal(this.app);
  fuzzy.setPlaceholder("Type something");
  log("SuggestModal / FuzzySuggestModal", "constructed; no native picker");

  var host = window.activeDocument.createElement("div");
  obsidian.MarkdownRenderer.renderMarkdown("**stub**", host, "", this)
    .then(function () {
      log("MarkdownRenderer", "resolved without touching the native preview");
    })
    .catch(function (err) { fail("MarkdownRenderer", err); });
  var popover = new obsidian.HoverPopover(this, host);
  popover.hoverEl.setText("Hover text (stub)");
  log("HoverPopover", "constructed in the plugin page");

  log("activeWindow", String(window.activeWindow === window));
  self.record();
  say("DOM samples built; see the plugin log");
};

DemoFull.prototype.showEditorExtensions = function () {
  log("editor.extension", this.counts.extensions + " extensions and " +
    this.counts.suggests + " suggests recorded (no CM6 editor)");
  this.record();
  say("Editor extensions are recorded but not applied");
};

DemoFull.prototype.registerSurfaces = function () {
  var self = this;

  this.registerView("demo-full-view", function (leaf) {
    return new DemoView(leaf, self);
  });
  this.addSettingTab(new DemoSettingTab(this.app, this));

  // Markdown processors are recorded; the native preview is not DOM.
  this.registerMarkdownPostProcessor(function (el, ctx) {
    log("registerMarkdownPostProcessor", "recorded only");
  });
  this.counts.processors += 1;
  this.registerMarkdownCodeBlockProcessor("demo", function (source, el) {
    el.setText("[demo] " + String(source || ""));
  });
  this.counts.processors += 1;

  obsidian.addIcon("demo-full-badge",
    '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">' +
    '<circle cx="12" cy="12" r="9" fill="#2d6a4f"/></svg>');
  var iconHost = window.activeDocument.createElement("div");
  obsidian.setIcon(iconHost, "demo-full-badge");
  log("addIcon / setIcon", "custom icon registered in the plugin page");
  log("activeDocument", String(window.activeDocument === window.activeWindow.document));
};

DemoFull.prototype.registerExtensionsAndStubs = function () {
  var self = this;

  // CodeMirror 6 extensions have no editor to attach to; the shim records
  // them so a plugin can still inspect its own registrations.
  this.registerEditorExtension({ name: "demo-full-extension" });
  this.counts.extensions += 1;
  var suggest = new obsidian.EditorSuggest(this.app);
  this.registerEditorSuggest(suggest);
  this.counts.suggests += 1;
  log("registerEditorExtension", "recorded; Xinkgo's editor is not CM6");

  this.registerObsidianProtocolHandler("demo-full", function (params) {
    say("Protocol action: " + JSON.stringify(params || {}));
  });
  this.counts.protocol += 1;
  this.registerExtensions(["demo"], "markdown");
  this.registerHoverLinkSource("demo-full",
    { display: "Demo Full", defaultMod: true });
  this.registerBasesView("demo-full", {});
  this.registerCliHandler();
  log("stubs", "protocol / extensions / hover / bases / cli are recorded only");

  // secretStorage is an in-page stub: it never reaches the OS keychain.
  this.app.secretStorage.setSecret("demo-full", "not-a-real-secret")
    .then(function () { return self.app.secretStorage.getSecret("demo-full"); })
    .then(function (value) {
      log("secretStorage", "returned " + String(value));
    })
    .catch(function (err) { fail("secretStorage", err); });
};

DemoFull.prototype.registerAppEvents = function () {
  var self = this;
  this.registerEvent(this.app.workspace.on("file-open", function (file) {
    var path = file && file.path ? file.path : String(file || "");
    self.state.lastPath = path;
    self.saveData(self.state).catch(function (err) { fail("saveData", err); });
    log("file-open", path);
  }));
  this.registerEvent(this.app.vault.on("modify", function (file) {
    log("vault modify", file && file.path ? file.path : "");
  }));
  this.registerEvent(this.app.metadataCache.on("changed", function (file) {
    log("metadata changed", file && file.path ? file.path : "");
  }));
  this.registerDomEvent(window.activeDocument, "click", function () {
    log("registerDomEvent", "click on the plugin page");
  });
  var ticks = 0;
  this.registerInterval(window.setInterval(function () {
    ticks += 1;
    if (ticks <= 3) log("registerInterval", "tick " + ticks);
  }, 60000));
  log("events.app", "file-open / vault / metadata / dom / interval registered");
};

DemoFull.prototype.exerciseStubs = function () {
  // compat.stub: a Node built-in resolves to an inert Proxy only while the
  // capability is granted. Without it the shim throws XinkgoCapabilityError.
  try {
    var pathModule = require("path");
    var joined = pathModule.join("a", "b");
    log("compat.stub", "path.join returned an inert " +
      typeof joined + " stub");
  } catch (err) {
    log("compat.stub", "not granted: " +
      String((err && err.message) || err));
  }
  log("registerEditorExtension", this.counts.extensions + " recorded");
  log("registerEditorSuggest", this.counts.suggests + " recorded");
  log("registerMarkdownPostProcessor", this.counts.processors + " recorded");
  log("registerObsidianProtocolHandler", this.counts.protocol + " recorded");
  this.record();
  say("Compatibility stubs exercised; see the plugin log");
};

module.exports = DemoFull;
