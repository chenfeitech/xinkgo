"use strict";

// Xinkgo plugin demo: the smallest useful plugin.
//
// Shape the host expects:
//   - a CommonJS bundle (no import/export statements)
//   - module.exports = one class extending obsidian.Plugin
//   - onload() registers everything; onunload() cleans up
//
// This demo deliberately stays on C1/C2: core API plus the native Vault /
// Editor / ribbon adapters. It never touches the plugin's own DOM surface,
// so the compatibility report is not degraded. Every command writes a line
// to the host log as well as showing a Notice, so the plugin is observable
// from the terminal or xinkgo.log without opening a window.

var obsidian = require("obsidian");

var PREFIX = "Xinkgo Demo: Basic";

// console.log is bridged to the host as a "plugin console" log line.
function log(what, detail) {
  console.log(PREFIX + " | " + what + " | " + String(detail || ""));
}

function activeEditor(app) {
  var view = app.workspace.getActiveViewOfType(obsidian.MarkdownView);
  return view ? view.editor : null;
}

var DemoBasic = (function () {
  function DemoBasic(app, manifest) {
    obsidian.Plugin.call(this, app, manifest);
    this.state = { runs: 0 };
    this.status = null;
  }
  DemoBasic.prototype = Object.create(obsidian.Plugin.prototype);
  DemoBasic.prototype.constructor = DemoBasic;

  // loadData/saveData persist data.json inside the plugin's own folder.
  DemoBasic.prototype.onload = function () {
    var self = this;
    return this.loadData().then(function (saved) {
      if (saved && typeof saved.runs === "number") {
        self.state.runs = saved.runs;
      }
      self.status = self.addStatusBarItem();
      self.refreshStatus();

      // A command is the basic unit of work: it appears in the command
      // palette (Cmd/Ctrl+P) with the plugin's name in front of it.
      self.addCommand({
        id: "hello",
        name: "Say hello",
        callback: function () {
          self.say("Hello from Xinkgo Demo: Basic");
        }
      });

      // Reading the active note through the Vault adapter.
      self.addCommand({
        id: "read-note",
        name: "Read the active note",
        callback: function () {
          self.readNote();
        }
      });

      // Reading and writing the editor buffer.
      self.addCommand({
        id: "count-words",
        name: "Count words in the active note",
        callback: function () {
          self.countWords();
        }
      });
      self.addCommand({
        id: "insert-date",
        name: "Insert today's date at the caret",
        editorCallback: function (editor) {
          editor.replaceSelection(obsidian.moment().format("YYYY-MM-DD"));
          self.say("Inserted a date");
        }
      });

      // checkCallback decides whether a command is offered at all. The
      // palette asks with checking=true before drawing the row.
      self.addCommand({
        id: "selection-only",
        name: "Describe the selection (needs a selection)",
        editorCheckCallback: function (checking, editor) {
          if (!editor.somethingSelected()) return false;
          if (checking) return true;
          self.say("Selected: " + editor.getSelection().slice(0, 40));
          return true;
        }
      });

      // One ribbon icon on the left strip, and a workspace event. Host
      // events carry no file object, so read the active note when needed.
      self.addRibbonIcon("calendar-days", "Demo Basic: Say hello",
        function () {
          self.say("Hello from the ribbon");
        });
      self.registerEvent(self.app.workspace.on("file-open",
        function () {
          var active = self.app.workspace.getActiveFile();
          log("file-open", active ? active.path : "(no active note)");
        }));

      new obsidian.Notice("Xinkgo Demo: Basic loaded", 3000);
      log("onload", "loaded; " + self.state.runs + " runs so far");
    }).catch(function (err) {
      console.error("Demo Basic onload", err);
    });
  };

  DemoBasic.prototype.onunload = function () {
    log("onunload", "cleaned up");
  };

  DemoBasic.prototype.refreshStatus = function () {
    if (this.status) {
      this.status.setText("Demo Basic: " + this.state.runs + " runs");
    }
  };

  DemoBasic.prototype.record = function () {
    this.state.runs += 1;
    this.saveData(this.state).catch(function (err) {
      console.error("Demo Basic saveData", err);
    });
    this.refreshStatus();
  };

  DemoBasic.prototype.say = function (text) {
    this.record();
    log("notice", text);
    new obsidian.Notice(text + " (run #" + this.state.runs + ")", 4000);
  };

  DemoBasic.prototype.readNote = function () {
    var self = this;
    var file = this.app.workspace.getActiveFile();
    if (!file) {
      new obsidian.Notice("Open a note first");
      return;
    }
    this.app.vault.read(file).then(function (text) {
      self.say(file.path + ": " + String(text).length + " characters");
    }).catch(function (err) {
      log("vault.read", "failed: " + String(err));
      new obsidian.Notice("vault.read failed: " + String(err));
    });
  };

  DemoBasic.prototype.countWords = function () {
    var editor = activeEditor(this.app);
    if (!editor) {
      new obsidian.Notice("Open a note first");
      return;
    }
    var text = editor.getValue().trim();
    var words = text ? text.split(/\s+/).length : 0;
    this.say(words + " words in the active note");
  };

  return DemoBasic;
})();

module.exports = DemoBasic;
