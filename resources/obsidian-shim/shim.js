(function () {
  "use strict";

  var cfg = window.__XINKGO__ || {};
  if (document.body) document.body.dataset.pluginId = cfg.pluginID || "";

  function CapabilityError(message) {
    this.name = "XinkgoCapabilityError";
    this.message = message;
    this.stack = new Error(message).stack;
  }
  CapabilityError.prototype = Object.create(Error.prototype);
  CapabilityError.prototype.constructor = CapabilityError;
  window.XinkgoCapabilityError = CapabilityError;

  var seq = 0;
  function rpc(method, params) {
    seq += 1;
    var body = {
      id: seq,
      method: method,
      params: params || {},
      token: cfg.token || ""
    };
    return fetch("/rpc", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body)
    }).then(function (response) {
      return response.json();
    }).then(function (reply) {
      if (!reply.ok) throw new Error(reply.error || "plugin bridge error");
      return reply.result;
    });
  }
  function fire(method, params) {
    rpc(method, params).catch(function (err) {
      console.error("plugin bridge", method, err);
    });
  }

  var state = {
    apiVersion: cfg.apiVersion || "1.0.0",
    hasActiveNote: false,
    activePath: "",
    editorText: "",
    selectionStart: 0,
    selectionEnd: 0,
    cursorLine: 0,
    cursorCh: 0,
    files: [],
    theme: "",
    language: "en",
    platform: "unknown",
    revision: 0
  };

  function refreshState() {
    return fetch("/state", { headers: { "Cache-Control": "no-store" } })
      .then(function (response) {
        if (!response.ok) throw new Error("state request failed");
        return response.json();
      }).then(function (next) {
        if (next && typeof next === "object") {
          state = next;
          if (!state.files) state.files = [];
        }
        return state;
      });
  }

  function offsetOf(text, pos) {
    var lines = String(text).split("\n");
    var line = Math.max(0, Math.min(lines.length - 1, Number(pos.line) || 0));
    var ch = Math.max(0, Math.min(lines[line].length, Number(pos.ch) || 0));
    var offset = 0;
    for (var i = 0; i < line; i++) offset += lines[i].length + 1;
    return offset + ch;
  }
  function posOf(text, offset) {
    var value = Math.max(0, Math.min(String(text).length, Number(offset) || 0));
    var before = String(text).slice(0, value);
    var parts = before.split("\n");
    return { line: parts.length - 1, ch: parts[parts.length - 1].length };
  }

  function writeEditor(text, selectionStart, selectionEnd) {
    var size = String(text).length;
    var start = Math.max(0, Math.min(size, Number(selectionStart) || 0));
    var end = Math.max(0, Math.min(size, Number(selectionEnd) || start));
    state.editorText = String(text);
    state.selectionStart = start;
    state.selectionEnd = end;
    var cursor = posOf(state.editorText, end);
    state.cursorLine = cursor.line;
    state.cursorCh = cursor.ch;
    fire("editor.write", {
      text: state.editorText,
      selectionStart: start,
      selectionEnd: end
    });
    return state.editorText;
  }

  function installDomHelpers(proto) {
    if (!proto || proto.__xinkgoHelpers) return;
    proto.__xinkgoHelpers = true;
    proto.setText = function (text) {
      this.textContent = text == null ? "" : String(text);
      return this;
    };
    proto.empty = function () {
      while (this.firstChild) this.removeChild(this.firstChild);
      return this;
    };
    proto.createEl = function (tag, attrs, text) {
      var el = document.createElement(tag || "div");
      if (attrs) {
        if (attrs.cls) el.className = attrs.cls;
        if (attrs.text) el.textContent = attrs.text;
        if (attrs.attr) for (var k in attrs.attr) el.setAttribute(k, attrs.attr[k]);
        if (attrs.title) el.setAttribute("title", attrs.title);
        if (attrs.href) el.setAttribute("href", attrs.href);
      }
      if (text != null) el.textContent = String(text);
      this.appendChild(el);
      return el;
    };
    proto.createDiv = function (attrs, text) {
      return this.createEl("div", attrs, text);
    };
    proto.createSpan = function (attrs, text) {
      return this.createEl("span", attrs, text);
    };
    proto.addClass = function () {
      for (var i = 0; i < arguments.length; i++) this.classList.add(arguments[i]);
      return this;
    };
    proto.removeClass = function () {
      for (var i = 0; i < arguments.length; i++) this.classList.remove(arguments[i]);
      return this;
    };
    proto.toggleClass = function (name, on) {
      if (on === undefined) this.classList.toggle(name);
      else this.classList.toggle(name, !!on);
      return this;
    };
    proto.setAttr = function (name, value) {
      this.setAttribute(name, String(value));
      return this;
    };
    proto.setAttrs = function (attrs) {
      for (var k in attrs) this.setAttribute(k, String(attrs[k]));
      return this;
    };
    proto.appendText = function (text) {
      this.appendChild(document.createTextNode(String(text)));
      return this;
    };
    proto.detach = function () {
      if (this.parentNode) this.parentNode.removeChild(this);
    };
    proto.find = function (selector) { return this.querySelector(selector); };
    proto.findAll = function (selector) {
      return Array.prototype.slice.call(this.querySelectorAll(selector));
    };
  }
  installDomHelpers(window.HTMLElement && window.HTMLElement.prototype);
  installDomHelpers(window.DocumentFragment && window.DocumentFragment.prototype);
  installDomHelpers(window.Element && window.Element.prototype);
  window.createFragment = function () {
    var fragment = document.createDocumentFragment();
    installDomHelpers(Object.getPrototypeOf(fragment));
    return fragment;
  };
  window.activeDocument = document;
  window.activeWindow = window;

  function Events() { this._listeners = {}; }
  Events.prototype.on = function (name, callback, ctx) {
    var list = this._listeners[name] || (this._listeners[name] = []);
    var entry = { fn: callback, ctx: ctx, ref: null };
    var self = this;
    entry.ref = { off: function () { self.offref(entry.ref); } };
    list.push(entry);
    return entry.ref;
  };
  Events.prototype.off = function (name, callback) {
    var list = this._listeners[name];
    if (!list) return;
    this._listeners[name] = list.filter(function (entry) {
      return entry.fn !== callback;
    });
  };
  Events.prototype.offref = function (ref) {
    for (var name in this._listeners) {
      this._listeners[name] = this._listeners[name].filter(function (entry) {
        return entry.ref !== ref;
      });
    }
  };
  Events.prototype.trigger = function (name) {
    var args = Array.prototype.slice.call(arguments, 1);
    var list = (this._listeners[name] || []).slice();
    for (var i = 0; i < list.length; i++) {
      list[i].fn.apply(list[i].ctx || this, args);
    }
  };
  Events.prototype.tryTrigger = function (ref, args) {
    for (var name in this._listeners) {
      var list = this._listeners[name];
      for (var i = 0; i < list.length; i++) {
        if (list[i].ref === ref) {
          list[i].fn.apply(list[i].ctx || this, args || []);
          return;
        }
      }
    }
  };

  function Component() {
    this._cleanups = [];
    this._children = [];
    this._loaded = false;
    this._unloaded = false;
  }
  Component.prototype.load = function () {
    if (this._loaded) return;
    this._loaded = true;
    return this.onload();
  };
  Component.prototype.onload = function () {};
  Component.prototype.unload = function () {
    if (this._unloaded) return;
    this._unloaded = true;
    try { this.onunload(); } finally {
      for (var i = 0; i < this._children.length; i++) this._children[i].unload();
      for (var j = this._cleanups.length - 1; j >= 0; j--) {
        try { this._cleanups[j](); } catch (err) { console.error(err); }
      }
      this._children = [];
      this._cleanups = [];
    }
  };
  Component.prototype.onunload = function () {};
  Component.prototype.addChild = function (child) {
    this._children.push(child);
    if (this._loaded && child.load) child.load();
    return child;
  };
  Component.prototype.removeChild = function (child) {
    child.unload();
    this._children = this._children.filter(function (one) { return one !== child; });
    return child;
  };
  Component.prototype.register = function (cb) {
    this._cleanups.push(cb);
    return cb;
  };
  Component.prototype.registerEvent = function (ref) {
    this._cleanups.push(function () { if (ref && ref.off) ref.off(); });
    return ref;
  };
  Component.prototype.registerDomEvent = function (el, type, callback, options) {
    if (!el || !el.addEventListener) return;
    el.addEventListener(type, callback, options);
    this._cleanups.push(function () { el.removeEventListener(type, callback, options); });
  };
  Component.prototype.registerInterval = function (id) {
    this._cleanups.push(function () { clearInterval(id); });
    return id;
  };

  function TAbstractFile(path, folder) {
    this.path = String(path || "");
    this.name = this.path.split("/").pop() || "/";
    this.parent = null;
    this.vault = null;
    this._folder = !!folder;
  }
  function TFile(record) {
    TAbstractFile.call(this, record && record.path, false);
    var name = this.name;
    var dot = name.lastIndexOf(".");
    this.extension = dot > 0 ? name.slice(dot + 1) : "";
    this.basename = dot > 0 ? name.slice(0, dot) : name;
    this.stat = {
      ctime: (record && record.ctime) || 0,
      mtime: (record && record.mtime) || 0,
      size: (record && record.size) || 0
    };
  }
  TFile.prototype = Object.create(TAbstractFile.prototype);
  TFile.prototype.constructor = TFile;
  function TFolder(path) {
    TAbstractFile.call(this, path, true);
    this.children = [];
  }
  TFolder.prototype = Object.create(TAbstractFile.prototype);
  TFolder.prototype.constructor = TFolder;
  TFolder.prototype.isRoot = function () {
    return this.path === "" || this.path === "/" || this.path === ".";
  };

  function filePath(file) {
    if (!file) return "";
    if (typeof file === "string") return file;
    return file.path || "";
  }
  function binaryBody(data) {
    if (data instanceof ArrayBuffer) {
      return { base64: arrayBufferToBase64(data) };
    }
    return { base64: String(data || "") };
  }
  function readBinary(path) {
    return rpc("vault.readBinary", { path: String(path || "") })
      .then(function (result) {
        return base64ToArrayBuffer(result && result.base64 || "");
      });
  }
  function fileRecord(path) {
    var wanted = normalizePath(path);
    for (var i = 0; i < state.files.length; i++) {
      if (normalizePath(state.files[i].path) === wanted) return state.files[i];
    }
    return null;
  }
  function fileObject(path) {
    var wanted = normalizePath(path);
    if (wanted === "" || wanted === "." || wanted === "/") return rootFolder();
    var record = fileRecord(wanted);
    if (!record) return null;
    var obj = record.folder ? new TFolder(record.path) : new TFile(record);
    obj.vault = vault;
    return obj;
  }
  function rootFolder() {
    var root = new TFolder("");
    root.vault = vault;
    var seen = {};
    for (var i = 0; i < state.files.length; i++) {
      var record = state.files[i];
      if (!record.folder) continue;
      var path = normalizePath(record.path);
      if (!path || seen[path]) continue;
      seen[path] = true;
      var folder = new TFolder(path);
      folder.vault = vault;
      root.children.push(folder);
    }
    for (var j = 0; j < state.files.length; j++) {
      var item = state.files[j];
      if (item.folder) continue;
      var file = new TFile(item);
      file.vault = vault;
      file.parent = root;
      root.children.push(file);
    }
    return root;
  }

  function Editor() {}
  Editor.prototype.getDoc = function () { return this; };
  Editor.prototype.refresh = function () {};
  Editor.prototype.focus = function () {};
  Editor.prototype.getValue = function () { return state.editorText || ""; };
  Editor.prototype.setValue = function (text) { return writeEditor(text, 0, 0); };
  Editor.prototype.getLine = function (n) {
    var lines = (state.editorText || "").split("\n");
    return lines[n] || "";
  };
  Editor.prototype.lineCount = function () {
    return (state.editorText || "").split("\n").length;
  };
  Editor.prototype.lastLine = function () { return this.lineCount() - 1; };
  Editor.prototype.setLine = function (n, text) {
    var lines = (state.editorText || "").split("\n");
    if (n < 0 || n >= lines.length) return;
    lines[n] = String(text);
    return writeEditor(lines.join("\n"), state.selectionStart, state.selectionEnd);
  };
  Editor.prototype.getCursor = function (which) {
    var offset = which === "anchor" ? state.selectionStart : state.selectionEnd;
    return posOf(state.editorText || "", offset);
  };
  Editor.prototype.setCursor = function (line, ch) {
    var pos = typeof line === "object" ? line : { line: line, ch: ch || 0 };
    var offset = offsetOf(state.editorText || "", pos);
    return writeEditor(state.editorText || "", offset, offset);
  };
  Editor.prototype.getSelection = function () {
    var start = Math.min(state.selectionStart, state.selectionEnd);
    var end = Math.max(state.selectionStart, state.selectionEnd);
    return (state.editorText || "").slice(start, end);
  };
  Editor.prototype.somethingSelected = function () {
    return state.selectionStart !== state.selectionEnd;
  };
  Editor.prototype.listSelections = function () {
    return [{
      anchor: posOf(state.editorText || "", state.selectionStart),
      head: posOf(state.editorText || "", state.selectionEnd)
    }];
  };
  Editor.prototype.getRange = function (from, to) {
    var a = offsetOf(state.editorText || "", from || { line: 0, ch: 0 });
    var b = offsetOf(state.editorText || "", to || from || { line: 0, ch: 0 });
    return (state.editorText || "").slice(Math.min(a, b), Math.max(a, b));
  };
  Editor.prototype.replaceRange = function (text, from, to) {
    var a = offsetOf(state.editorText || "", from || { line: 0, ch: 0 });
    var b = to ? offsetOf(state.editorText || "", to) : a;
    var before = (state.editorText || "").slice(0, Math.min(a, b));
    var after = (state.editorText || "").slice(Math.max(a, b));
    var value = String(text == null ? "" : text);
    return writeEditor(before + value + after, before.length + value.length,
      before.length + value.length);
  };
  Editor.prototype.replaceSelection = function (text) {
    var start = Math.min(state.selectionStart, state.selectionEnd);
    var end = Math.max(state.selectionStart, state.selectionEnd);
    var value = String(text == null ? "" : text);
    var before = (state.editorText || "").slice(0, start);
    var after = (state.editorText || "").slice(end);
    return writeEditor(before + value + after, before.length + value.length,
      before.length + value.length);
  };
  Editor.prototype.setSelection = function (anchor, head) {
    var a = offsetOf(state.editorText || "", anchor);
    var b = offsetOf(state.editorText || "", head);
    return writeEditor(state.editorText || "", a, b);
  };
  Editor.prototype.posToOffset = function (pos) {
    return offsetOf(state.editorText || "", pos);
  };
  Editor.prototype.offsetToPos = function (offset) {
    return posOf(state.editorText || "", offset);
  };
  Editor.prototype.transaction = function (tx) {
    var changes = tx && tx.changes;
    if (Array.isArray(changes)) {
      for (var i = 0; i < changes.length; i++) {
        var one = changes[i];
        if (one && one.text !== undefined && one.from && one.to) {
          this.replaceRange(one.text, one.from, one.to);
        }
      }
    }
    if (tx && tx.selection) this.setSelection(tx.selection.anchor, tx.selection.head);
  };

  function MarkdownView() {
    this.app = app;
    this.editor = new Editor();
    this.file = workspace.getActiveFile();
    this.leaf = null;
    this.hoverPopover = null;
  }
  MarkdownView.prototype.getViewType = function () { return "markdown"; };
  MarkdownView.prototype.getMode = function () { return "source"; };
  MarkdownView.prototype.getViewData = function () { return this.editor.getValue(); };
  MarkdownView.prototype.setViewData = function (data) { return this.editor.setValue(data); };
  MarkdownView.prototype.showSearch = function () {};
  MarkdownView.prototype.requestSave = function () {};
  MarkdownView.prototype.clear = function () {};

  function Workspace() {
    Events.call(this);
    this.layoutReady = true;
    this.activeLeaf = null;
  }
  Workspace.prototype = Object.create(Events.prototype);
  Workspace.prototype.constructor = Workspace;
  Object.defineProperty(Workspace.prototype, "activeEditor", {
    get: function () { return state.hasActiveNote ? new MarkdownView() : null; }
  });
  Workspace.prototype.getActiveFile = function () {
    return state.hasActiveNote ? fileObject(state.activePath) : null;
  };
  Workspace.prototype.getActiveViewOfType = function (type) {
    if (!state.hasActiveNote) return null;
    if (type === MarkdownView || type === "markdown" ||
        (type && type.prototype && type.prototype instanceof MarkdownView)) {
      return new MarkdownView();
    }
    return null;
  };
  Workspace.prototype.getLeaf = function () {
    throw new CapabilityError("WorkspaceLeaf is not supported by Xinkgo's native UI");
  };
  Workspace.prototype.openLinkText = function (linktext) {
    return rpc("workspace.open", { linktext: String(linktext || "") });
  };
  Workspace.prototype.onLayoutReady = function (callback) { if (callback) callback(); };
  Workspace.prototype.iterateAllLeaves = function (callback) {
    if (callback && state.hasActiveNote) callback({ view: new MarkdownView() });
  };
  Workspace.prototype.changeLayout = function () {
    throw new CapabilityError("changeLayout is not supported by Xinkgo's native UI");
  };
  Workspace.prototype.getLayout = function () { return {}; };

  function Vault() {
    Events.call(this);
    this.configDir = ".obsidian";
    this.adapter = {
      getName: function () { return state.vaultName || "vault"; },
      exists: function (path) { return rpc("vault.exists", { path: String(path || "") }); },
      stat: function (path) { return rpc("vault.stat", { path: String(path || "") }); },
      list: function (path) { return rpc("vault.list", { path: String(path || "") }); },
      read: function (path) { return rpc("vault.read", { path: String(path || "") }); },
      readBinary: function (path) { return readBinary(path); },
      write: function (path, data) { return rpc("vault.write", { path: String(path || ""), text: String(data) }); },
      writeBinary: function (path, data) { return rpc("vault.writeBinary", { path: String(path || ""), data: binaryBody(data).base64 }); },
      append: function (path, data) { return rpc("vault.append", { path: String(path || ""), text: String(data) }); },
      remove: function (path) { return rpc("vault.delete", { path: String(path || "") }); },
      mkdir: function (path) { return rpc("vault.createFolder", { path: String(path || "") }); },
      rmdir: function (path) { return rpc("vault.delete", { path: String(path || "") }); },
      rename: function (oldPath, newPath) { return rpc("vault.rename", { path: String(oldPath || ""), to: String(newPath || "") }); },
      trash: function (path) { return rpc("vault.delete", { path: String(path || "") }); },
      getResourcePath: function (path) { return "/vault/" + normalizePath(path); }
    };
  }
  Vault.prototype = Object.create(Events.prototype);
  Vault.prototype.constructor = Vault;
  Vault.prototype.getName = function () { return state.vaultName || "vault"; };
  Vault.prototype.getRoot = function () { return rootFolder(); };
  Vault.prototype.getAbstractFileByPath = function (path) { return fileObject(path); };
  Vault.prototype.getFileByPath = function (path) {
    var obj = fileObject(path);
    return obj instanceof TFile ? obj : null;
  };
  Vault.prototype.getFolderByPath = function (path) {
    var obj = fileObject(path);
    return obj instanceof TFolder ? obj : null;
  };
  Vault.prototype.read = function (file) { return rpc("vault.read", { path: filePath(file) }); };
  Vault.prototype.cachedRead = function (file) { return this.read(file); };
  Vault.prototype.readBinary = function (file) { return readBinary(filePath(file)); };
  Vault.prototype.modify = function (file, data) { return rpc("vault.write", { path: filePath(file), text: String(data) }); };
  Vault.prototype.modifyBinary = function (file, data) { return rpc("vault.writeBinary", { path: filePath(file), data: binaryBody(data).base64 }); };
  Vault.prototype.create = function (path, data) { return rpc("vault.create", { path: String(path || ""), text: String(data) }); };
  Vault.prototype.createBinary = function (path, data) { return rpc("vault.writeBinary", { path: String(path || ""), data: binaryBody(data).base64 }); };
  Vault.prototype.createFolder = function (path) { return rpc("vault.createFolder", { path: String(path || "") }); };
  Vault.prototype.append = function (file, data) { return rpc("vault.append", { path: filePath(file), text: String(data) }); };
  Vault.prototype.delete = function (file) { return rpc("vault.delete", { path: filePath(file) }); };
  Vault.prototype.trash = function (file) { return this.delete(file); };
  Vault.prototype.rename = function (file, newPath) { return rpc("vault.rename", { path: filePath(file), to: String(newPath || "") }); };
  Vault.prototype.getResourcePath = function (file) { return "/vault/" + normalizePath(filePath(file)); };

  function MetadataCache() { Events.call(this); }
  MetadataCache.prototype = Object.create(Events.prototype);
  MetadataCache.prototype.constructor = MetadataCache;
  MetadataCache.prototype.getFileCache = function () { return {}; };
  MetadataCache.prototype.getCache = function () { return {}; };
  MetadataCache.prototype.getFirstLinkpathDest = function (linkpath) {
    var wanted = normalizePath(String(linkpath || ""));
    for (var i = 0; i < state.files.length; i++) {
      if (!state.files[i].folder && normalizePath(state.files[i].path) === wanted) {
        return fileObject(wanted);
      }
    }
    return null;
  };
  MetadataCache.prototype.fileToLinktext = function (file, sourcePath) {
    var path = filePath(file);
    if (!path) return "";
    if (sourcePath && normalizePath(path).indexOf(normalizePath(sourcePath)) === 0) {
      return normalizePath(path).slice(normalizePath(sourcePath).length).replace(/^\//, "");
    }
    return path;
  };

  function FileManager() {}
  FileManager.prototype.renameFile = function (file, newPath) { return vault.rename(file, newPath); };
  FileManager.prototype.generateMarkdownLink = function (file, sourcePath, subpath, alias) {
    var path = filePath(file);
    var text = alias || (path.split("/").pop() || "").replace(/\.md$/, "");
    return "[[" + path + (subpath || "") + "|" + text + "]]";
  };
  FileManager.prototype.processFrontMatter = function (file, fn) {
    return vault.read(file).then(function (text) {
      var front = {};
      if (fn) fn(front);
      return vault.modify(file, text);
    });
  };

  function Notice(message, duration) {
    var text = message && message.textContent !== undefined ?
      message.textContent : String(message == null ? "" : message);
    this.messageEl = document.createElement("div");
    this.noticeEl = this.messageEl;
    this.containerEl = document.createElement("div");
    this.messageEl.setText(text);
    this.duration = duration === undefined ? 5000 : duration;
    fire("notice", { text: text, duration: this.duration });
  }
  Notice.prototype.setMessage = function (message) {
    var text = message && message.textContent !== undefined ?
      message.textContent : String(message == null ? "" : message);
    this.messageEl.setText(text);
    fire("notice", { text: text, duration: this.duration });
    return this;
  };
  Notice.prototype.hide = function () { this.messageEl.detach(); };

  function Scope() {}
  Scope.prototype.register = function () { return {}; };
  Scope.prototype.unregister = function () {};

  var commands = {};
  var actions = {};
  var plugin = null;
  function editorForCommand() { return new Editor(); }
  function viewForCommand() { return new MarkdownView(); }
  function commandVisible(entry) {
    var command = entry.command;
    try {
      if (command.editorCheckCallback) {
        return !!command.editorCheckCallback(true, editorForCommand(), viewForCommand());
      }
      if (command.checkCallback) return !!command.checkCallback(true);
    } catch (err) {
      console.error("checkCallback", err);
      return false;
    }
    return true;
  }
  function updateVisibility() {
    var updates = [];
    for (var id in commands) {
      updates.push({ id: id, visible: commandVisible(commands[id]) });
    }
    if (updates.length > 0) fire("command.visibility", { updates: updates });
  }

  function Plugin(pluginApp, manifest) {
    Component.call(this);
    this.app = pluginApp;
    this.manifest = manifest || cfg.manifest || {};
    this.settings = undefined;
    this._actionSeq = 0;
  }
  Plugin.prototype = Object.create(Component.prototype);
  Plugin.prototype.constructor = Plugin;
  Plugin.prototype.onload = function () {};
  Plugin.prototype.onunload = function () {};
  Plugin.prototype.loadData = function () { return rpc("data.load", {}); };
  Plugin.prototype.saveData = function (data) { return rpc("data.save", { data: data }); };
  Plugin.prototype.addCommand = function (command) {
    var local = command && command.id ? String(command.id) : "";
    if (!local) throw new Error("a command needs an id");
    var id = String(cfg.pluginID || "plugin") + ":" + local;
    var prefixed = {};
    for (var key in command) prefixed[key] = command[key];
    prefixed.id = id;
    prefixed.name = String(cfg.pluginName || cfg.pluginID || "Plugin") +
      ": " + String(command.name || local);
    var entry = { command: prefixed, plugin: this };
    commands[id] = entry;
    fire("command.add", {
      id: id, name: prefixed.name,
      visible: commandVisible(entry), icon: command.icon || ""
    });
    return prefixed;
  };
  Plugin.prototype.removeCommand = function (commandId) {
    var id = String(cfg.pluginID || "plugin") + ":" + String(commandId || "");
    delete commands[id];
    fire("command.remove", { id: id });
  };
  Plugin.prototype.addRibbonIcon = function (icon, title, callback) {
    this._actionSeq += 1;
    var id = "ribbon:" + this._actionSeq;
    actions[id] = callback;
    var el = document.createElement("div");
    el.className = "xinkgo-plugin-ribbon";
    el.setAttr("data-icon", icon || "");
    el.setText(title || "");
    fire("action.add", { id: id, name: String(title || "Plugin action"), kind: "ribbon" });
    return el;
  };
  Plugin.prototype.addStatusBarItem = function () {
    var el = document.createElement("span");
    el.className = "xinkgo-plugin-status";
    el.setText = function (text) {
      fire("status.set", { text: String(text == null ? "" : text) });
      return this;
    };
    fire("status.add", {});
    return el;
  };
  Plugin.prototype.addSettingTab = function () {
    throw new CapabilityError("PluginSettingTab needs a plugin DOM surface (C3)");
  };
  Plugin.prototype.registerView = function () {
    throw new CapabilityError("WorkspaceLeaf/ItemView is not supported by Xinkgo's native UI");
  };
  Plugin.prototype.registerMarkdownPostProcessor = function () {
    throw new CapabilityError("Markdown post processing needs an HTML preview surface (C3)");
  };
  Plugin.prototype.registerMarkdownCodeBlockProcessor = function () {
    throw new CapabilityError("Markdown code block processing needs an HTML preview surface (C3)");
  };
  Plugin.prototype.registerEditorExtension = function () {
    throw new CapabilityError("CodeMirror 6 extensions are not supported (C4)");
  };
  Plugin.prototype.registerEditorSuggest = function () {
    throw new CapabilityError("EditorSuggest is not supported (C4)");
  };
  Plugin.prototype.registerObsidianProtocolHandler = function () {
    throw new CapabilityError("obsidian:// handlers are not supported in the P0 spike");
  };
  Plugin.prototype.registerCliHandler = function () {
    throw new CapabilityError("CLI handlers are not supported in the P0 spike");
  };
  Plugin.prototype.registerHoverLinkSource = function () {
    throw new CapabilityError("registerHoverLinkSource is not supported in the P0 spike");
  };
  Plugin.prototype.registerBasesView = function () {
    throw new CapabilityError("registerBasesView is not supported in the P0 spike");
  };
  Plugin.prototype.registerExtensions = function () {
    throw new CapabilityError("registerExtensions is not supported in the P0 spike");
  };
  Plugin.prototype.onUserEnable = function () {};
  Plugin.prototype.onExternalSettingsChange = function () {};

  var vault = new Vault();
  var workspace = new Workspace();
  var metadataCache = new MetadataCache();
  var fileManager = new FileManager();
  var app = {
    vault: vault,
    workspace: workspace,
    metadataCache: metadataCache,
    fileManager: fileManager,
    keymap: {
      isModifierEvent: function () { return false; },
      getRootScope: function () { return new Scope(); }
    },
    scope: new Scope(),
    renderContext: {},
    lastEvent: null,
    isDarkMode: function () { return !!state.dark; },
    loadLocalStorage: function (key) {
      var raw = window.localStorage.getItem("xinkgo:" + String(key));
      return raw === null ? null : JSON.parse(raw);
    },
    saveLocalStorage: function (key, data) {
      if (data === null) window.localStorage.removeItem("xinkgo:" + String(key));
      else window.localStorage.setItem("xinkgo:" + String(key), JSON.stringify(data));
    }
  };
  Object.defineProperty(app, "secretStorage", {
    get: function () {
      throw new CapabilityError("secretStorage is not supported in the P0 spike");
    }
  });

  function normalizePath(path) {
    var value = String(path || "").replace(/\\/g, "/").replace(/^\/+/, "");
    var parts = [];
    var split = value.split("/");
    for (var i = 0; i < split.length; i++) {
      var one = split[i];
      if (one === "" || one === ".") continue;
      if (one === "..") { if (parts.length > 0) parts.pop(); continue; }
      parts.push(one);
    }
    return parts.join("/");
  }
  function getLinkpath(linktext) {
    return String(linktext || "").split("#")[0].split("|")[0].trim();
  }
  function sanitizeHTMLToDom(html) {
    return document.createRange().createContextualFragment(String(html || ""));
  }
  function htmlToMarkdown(html) {
    var div = document.createElement("div");
    div.innerHTML = String(html || "");
    return div.textContent || "";
  }
  function arrayBufferToBase64(buffer) {
    var bytes = new Uint8Array(buffer);
    var text = "";
    for (var i = 0; i < bytes.length; i++) text += String.fromCharCode(bytes[i]);
    return btoa(text);
  }
  function base64ToArrayBuffer(base64) {
    var text = atob(String(base64 || ""));
    var bytes = new Uint8Array(text.length);
    for (var i = 0; i < text.length; i++) bytes[i] = text.charCodeAt(i);
    return bytes.buffer;
  }
  function compareVersions(a, b) {
    var left = String(a || "0.0.0").split(/[-+]/)[0].split(".");
    var right = String(b || "0.0.0").split(/[-+]/)[0].split(".");
    for (var i = 0; i < 3; i++) {
      var x = Number(left[i] || 0), y = Number(right[i] || 0);
      if (x < y) return -1;
      if (x > y) return 1;
    }
    return 0;
  }
  function requireApiVersion(version) {
    return compareVersions(cfg.apiVersion || "1.0.0", version) >= 0;
  }
  function moment(input) {
    var date = input ? new Date(input) : new Date();
    var api = {
      toDate: function () { return new Date(date.getTime()); },
      valueOf: function () { return date.getTime(); },
      isValid: function () { return !isNaN(date.getTime()); },
      format: function (pattern) {
        var p = String(pattern || "YYYY-MM-DD");
        var pad = function (n) { return (n < 10 ? "0" : "") + n; };
        return p.replace(/YYYY/g, date.getFullYear())
          .replace(/MM/g, pad(date.getMonth() + 1))
          .replace(/DD/g, pad(date.getDate()))
          .replace(/HH/g, pad(date.getHours()))
          .replace(/mm/g, pad(date.getMinutes()))
          .replace(/ss/g, pad(date.getSeconds()));
      },
      add: function (n, unit) {
        if (unit === "days" || unit === "day" || unit === "d") date.setDate(date.getDate() + n);
        if (unit === "months" || unit === "month" || unit === "M") date.setMonth(date.getMonth() + n);
        if (unit === "years" || unit === "year" || unit === "y") date.setFullYear(date.getFullYear() + n);
        return api;
      },
      startOf: function () { return api; },
      endOf: function () { return api; },
      fromNow: function () { return ""; },
      locale: function () { return api; },
      unix: function () { return Math.floor(date.getTime() / 1000); }
    };
    return api;
  }
  moment.locale = function () { return "en"; };
  moment.localeData = function () { return { longDateFormat: function () { return "YYYY-MM-DD"; } }; };

  function requestUrl(request) {
    var req = typeof request === "string" ? { url: request } : (request || {});
    var body = { text: "" };
    if (req.body instanceof ArrayBuffer) {
      body = { base64: arrayBufferToBase64(req.body) };
    } else if (req.body != null) {
      body = { text: String(req.body) };
    }
    var promise = rpc("network.request", {
      url: String(req.url || ""),
      method: String(req.method || "GET"),
      headers: req.headers || {},
      body: body,
      throw: req.throw !== false
    }).then(function (result) {
      var arrayBuffer = result.base64 ? base64ToArrayBuffer(result.base64) :
        new ArrayBuffer(0);
      var json = null;
      try { json = JSON.parse(result.text || ""); } catch (err) { json = null; }
      var response = {
        status: result.status || 0,
        headers: result.headers || {},
        arrayBuffer: arrayBuffer,
        json: json,
        text: result.text || ""
      };
      response.arrayBuffer = Promise.resolve(arrayBuffer);
      response.json = Promise.resolve(json);
      response.text = Promise.resolve(response.text);
      return response;
    });
    promise.arrayBuffer = promise.then(function (r) { return r.arrayBuffer; });
    promise.json = promise.then(function (r) { return r.json; });
    promise.text = promise.then(function (r) { return r.text; });
    return promise;
  }

  var ua = navigator.userAgent || "";
  var Platform = {
    isDesktop: true, isMobile: false,
    isDesktopApp: true, isMobileApp: false,
    isMacOS: ua.indexOf("Mac") >= 0,
    isWin: ua.indexOf("Windows") >= 0,
    isLinux: ua.indexOf("Linux") >= 0,
    isIosApp: false, isAndroidApp: false,
    isSafari: ua.indexOf("Safari") >= 0 && ua.indexOf("Chrome") < 0,
    isFirefox: ua.indexOf("Firefox") >= 0,
    isConsoleMobile: false
  };

  function unsupportedClass(name) {
    return function () {
      throw new CapabilityError("unsupported obsidian export: " + name);
    };
  }
  function unsupported(name) { return unsupportedClass(name); }

  var exports = {
    App: app, Plugin: Plugin, Component: Component, Events: Events,
    Notice: Notice, TAbstractFile: TAbstractFile, TFile: TFile,
    TFolder: TFolder, Vault: Vault, Workspace: Workspace,
    MarkdownView: MarkdownView, Editor: Editor, MetadataCache: MetadataCache,
    FileManager: FileManager, Scope: Scope,
    Platform: Platform, apiVersion: cfg.apiVersion || "1.0.0",
    requireApiVersion: requireApiVersion, moment: moment,
    normalizePath: normalizePath, getLinkpath: getLinkpath,
    sanitizeHTMLToDom: sanitizeHTMLToDom, htmlToMarkdown: htmlToMarkdown,
    arrayBufferToBase64: arrayBufferToBase64,
    base64ToArrayBuffer: base64ToArrayBuffer, requestUrl: requestUrl,
    getLanguage: function () { return state.language || navigator.language || "en"; },
    getFrontMatterInfo: function () {
      return { exists: false, frontmatter: "", from: 0, to: 0 };
    },
    prepareSimpleSearch: function (query) {
      var q = String(query || "").toLowerCase();
      return function (text) {
        var at = String(text || "").toLowerCase().indexOf(q);
        return at < 0 ? null : { score: 0, matches: [[at, at + q.length]] };
      };
    },
    iterateRefs: function () { return false; },
    addIcon: unsupported("addIcon"), setIcon: unsupported("setIcon"),
    getIcon: unsupported("getIcon"), getIconIds: function () { return []; },
    loadMermaid: unsupported("loadMermaid"),
    loadMathJax: unsupported("loadMathJax"),
    Modal: unsupported("Modal"), SuggestModal: unsupported("SuggestModal"),
    FuzzySuggestModal: unsupported("FuzzySuggestModal"),
    Setting: unsupported("Setting"), SettingTab: unsupported("SettingTab"),
    PluginSettingTab: unsupported("PluginSettingTab"),
    Menu: unsupported("Menu"), MenuItem: unsupported("MenuItem"),
    MenuSeparator: unsupported("MenuSeparator"),
    ItemView: unsupported("ItemView"), View: unsupported("View"),
    WorkspaceLeaf: unsupported("WorkspaceLeaf"),
    EditorSuggest: unsupported("EditorSuggest"),
    MarkdownRenderer: unsupported("MarkdownRenderer"),
    MarkdownRenderChild: unsupported("MarkdownRenderChild"),
    HoverPopover: unsupported("HoverPopover"),
    AbstractInputSuggest: unsupported("AbstractInputSuggest")
  };

  window.module = { exports: {} };
  window.exports = window.module.exports;
  window.require = function require(name) {
    if (name === "obsidian") {
      return new Proxy(exports, {
        get: function (target, property) {
          if (property in target) return target[property];
          if (typeof property === "symbol") return undefined;
          return unsupported("obsidian." + String(property));
        }
      });
    }
    throw new CapabilityError("unsupported module: " + String(name));
  };
  window.obsidian = window.require("obsidian");

  var shim = {
    cfg: cfg, app: app, state: state, rpc: rpc, fire: fire,
    refreshState: refreshState, commands: commands, actions: actions,
    plugin: null,
    setPlugin: function (value) { plugin = value; shim.plugin = value; },
    getPlugin: function () { return plugin; },
    updateVisibility: updateVisibility,
    fail: function (text) { fire("error", { text: String(text || "plugin failed") }); },
    log: function (level, text) { fire("log", { level: level, text: String(text || "") }); },
    trigger: function (name, text) {
      var parts = String(name || "").split(".");
      var kind = parts[0];
      var args = text === undefined ? [] : [text];
      if (kind === "vault") vault.trigger.apply(vault, [parts.slice(1).join(".")].concat(args));
      else if (kind === "metadata") metadataCache.trigger.apply(metadataCache, [parts.slice(1).join(".")].concat(args));
      else workspace.trigger.apply(workspace, [String(name)].concat(args));
      workspace.trigger("stateChanged");
    },
    runCommand: function (id) {
      var entry = commands[id];
      if (!entry) { fire("error", { text: "unknown command: " + id }); return; }
      var command = entry.command;
      try {
        if (command.editorCallback) command.editorCallback(editorForCommand(), viewForCommand());
        else if (command.editorCheckCallback) command.editorCheckCallback(false, editorForCommand(), viewForCommand());
        else if (command.checkCallback) command.checkCallback(false);
        else if (command.callback) command.callback();
      } catch (err) {
        fire("error", { text: String((err && err.stack) || err) });
      }
    },
    runAction: function (id) {
      var callback = actions[id];
      if (!callback) return;
      try { callback({ preventDefault: function () {}, stopPropagation: function () {} }); }
      catch (err) { fire("error", { text: String((err && err.stack) || err) }); }
    },
    unload: function (requestID) {
      var work = Promise.resolve();
      if (plugin && plugin.unload) {
        try { work = Promise.resolve(plugin.unload()); } catch (err) { work = Promise.reject(err); }
      }
      return work.catch(function (err) {
        fire("error", { text: String((err && err.stack) || err) });
      }).then(function () {
        commands = {};
        actions = {};
        shim.commands = commands;
        shim.actions = actions;
        return rpc("unloadDone", { requestID: requestID || "" });
      });
    },
    boot: function () {
      var exported = window.module.exports;
      var Klass = exported && (exported.default || exported);
      if (typeof Klass !== "function") {
        return shim.fail("main.js has no default Plugin class");
      }
      var instance;
      try { instance = new Klass(app, cfg.manifest || {}); }
      catch (err) { return shim.fail(String((err && err.stack) || err)); }
      shim.setPlugin(instance);
      return shim.refreshState().then(function () {
        if (instance.onUserEnable) instance.onUserEnable();
        return instance.onload ? instance.onload() : undefined;
      }).then(function () {
        shim.updateVisibility();
        return rpc("ready", {});
      }).catch(function (err) {
        shim.fail(String((err && err.stack) || err));
      });
    }
  };
  window.__XINKGO_SHIM__ = shim;

  if (window.mygo && typeof window.mygo.on === "function") {
    window.mygo.on("xinkgo-plugin-event", function (event) {
      if (!event) return;
      if (event.type === "stateChanged") {
        shim.refreshState().then(shim.updateVisibility).catch(function (err) {
          console.error("state refresh", err);
        });
      } else if (event.type === "event") {
        shim.trigger(event.name, event.text);
      } else if (event.type === "command") {
        shim.runCommand(event.commandID);
      } else if (event.type === "action") {
        shim.runAction(event.actionID);
      } else if (event.type === "unload") {
        shim.unload(event.requestID);
      }
    });
  }

  ["log", "warn", "error"].forEach(function (level) {
    var original = console[level] ? console[level].bind(console) : function () {};
    console[level] = function () {
      var args = Array.prototype.slice.call(arguments);
      original.apply(console, args);
      var text = args.map(function (one) {
        if (one instanceof Error) return one.stack || one.message;
        try { return typeof one === "string" ? one : JSON.stringify(one); }
        catch (err) { return String(one); }
      }).join(" ");
      shim.log(level, text.slice(0, 4000));
    };
  });
  window.addEventListener("error", function (event) {
    shim.fail(String((event.error && event.error.stack) || event.message || "script error"));
  });
  window.addEventListener("unhandledrejection", function (event) {
    var reason = event.reason;
    shim.fail(String((reason && reason.stack) || reason || "unhandled rejection"));
  });
})();
