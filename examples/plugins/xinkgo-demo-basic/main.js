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
// so the compatibility report is not degraded.

var obsidian = require("obsidian");

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

      // One ribbon icon on the left strip, and a workspace event.
      self.addRibbonIcon("calendar-days", "Demo Basic: Say hello",
        function () {
          self.say("Hello from the ribbon");
        });
      self.registerEvent(self.app.workspace.on("file-open",
        function (path) {
          console.log("Demo Basic saw file-open: " + String(path || ""));
        }));

      new obsidian.Notice("Xinkgo Demo: Basic loaded", 3000);
    }).catch(function (err) {
      console.error("Demo Basic onload", err);
    });
  };

  DemoBasic.prototype.onunload = function () {
    console.log("Xinkgo Demo: Basic unloaded");
  };

  DemoBasic.prototype.refreshStatus = function () {
    if (this.status) {
      this.status.setText("Demo Basic: " + this.state.runs + " runs");
    }
  };

  DemoBasic.prototype.record = function () {
    this.state.runs += 1;
    this.saveData(this.state);
    this.refreshStatus();
  };

  DemoBasic.prototype.say = function (text) {
    this.record();
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
