package app

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	pathpkg "path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/plugin"
	"github.com/helightxu/xinkgo/internal/state"
	"github.com/helightxu/xinkgo/internal/view"
)

// pluginScheme is the custom origin of every plugin WebView. MyGo's
// untrusted-page rules keep the page from calling bound Go methods; the
// protocol handler is the only bridge.
const pluginScheme = "xinkgo-plugin"

const pluginEventName = "xinkgo-plugin-event"

const pluginCSP = "default-src 'none'; script-src 'self'; " +
	"style-src 'self' 'unsafe-inline'; img-src 'self' data:; " +
	"connect-src 'self'; font-src 'self'; object-src 'none'; " +
	"base-uri 'none'; form-action 'none'"

// pluginEvent is a small control message from Go to one plugin page.
// Large payloads go through /state or the custom scheme instead.
type pluginEvent struct {
	Type      string `json:"type"`
	Name      string `json:"name,omitempty"`
	CommandID string `json:"commandID,omitempty"`
	ActionID  string `json:"actionID,omitempty"`
	ViewType  string `json:"viewType,omitempty"`
	RequestID string `json:"requestID,omitempty"`
	Revision  uint64 `json:"revision,omitempty"`
	Text      string `json:"text,omitempty"`
}

var pluginEvents = mygo.NewEvent[pluginEvent](pluginEventName)

var (
	cssImportPattern = regexp.MustCompile(`(?i)@import[^;]*;?`)
	cssRemoteURL     = regexp.MustCompile(`(?i)url\(\s*['"]?https?://[^)]+\)`)
)

// pluginFile is one file or folder of the vault, as the JS shim sees it.
type pluginFile struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	Folder    bool   `json:"folder"`
	Extension string `json:"extension,omitempty"`
	Basename  string `json:"basename,omitempty"`
	Size      int64  `json:"size"`
	CTime     int64  `json:"ctime"`
	MTime     int64  `json:"mtime"`
}

// pluginStatePayload is the cached snapshot the plugin shim mirrors.
type pluginStatePayload struct {
	APIVersion     string       `json:"apiVersion"`
	HasActiveNote  bool         `json:"hasActiveNote"`
	ActivePath     string       `json:"activePath"`
	EditorText     string       `json:"editorText"`
	SelectionStart int          `json:"selectionStart"`
	SelectionEnd   int          `json:"selectionEnd"`
	CursorLine     int          `json:"cursorLine"`
	CursorCh       int          `json:"cursorCh"`
	Files          []pluginFile `json:"files"`
	VaultName      string       `json:"vaultName"`
	Theme          string       `json:"theme"`
	Language       string       `json:"language"`
	Platform       string       `json:"platform"`
	Dark           bool         `json:"dark"`
	Revision       uint64       `json:"revision"`
}

// pluginInstance is one enabled plugin's isolated runtime.
type pluginInstance struct {
	id       string
	host     string
	token    string
	dir      string
	source   string
	manifest plugin.Manifest
	report   plugin.Report
	grants   map[string]bool
	page     *mygo.WebView
	ready    bool
	failed   bool
	unloadID string
	timer    *time.Timer
	commands map[string]view.PluginCommand
	ribbon   map[string]view.PluginAction
	views    map[string]view.PluginView
}

// pluginUI is the state of the plugin settings pane. View functions read
// it under the host lock; they never do I/O themselves.
type pluginUI struct {
	tab           string
	query         string
	supportedOnly bool
	showDegraded  bool
	browseLimit   int
	lastQuery     string
	lastSupported bool
	status        string
	installed     []plugin.Report
	busy          map[string]bool
	// open is whether a plugin card's details and grants are unfolded.
	// It is pure UI state: not persisted, not read off the main thread.
	open map[string]bool
}

// pluginHost owns discovery, the custom protocol, WebView isolation and
// the explicit bridge between JS and the running app.
type pluginHost struct {
	st        *appState
	dir       string
	resources string
	userDir   string
	root      string

	mu             sync.RWMutex
	assets         map[string][]byte
	instances      map[string]*pluginInstance
	byPage         map[*mygo.Page]*pluginInstance
	byHost         map[string]*pluginInstance
	reports        map[string]plugin.Report
	library        plugin.Library
	libraryIDs     map[string]bool
	contracts      map[string]plugin.ContractResult
	saved          state.PluginState
	snapshot       pluginStatePayload
	revision       uint64
	catalog        []plugin.CatalogEntry
	compatible     []plugin.CompatibleEntry
	stats          map[string]plugin.PluginStats
	details        map[string]plugin.PluginDetail
	detailLoading  map[string]bool
	catalogLoaded  bool
	catalogLoading bool
	catalogRev     uint64
	catalogView    []plugin.BrowserItem
	catalogViewRev uint64
	coreCache      []plugin.BrowserItem
	coreRev        uint64
	browserKey     string
	browser        []plugin.BrowserItem
	ui             pluginUI
}

// newPluginHost registers the custom scheme before any window is made.
// All of the paths it needs are available only after WhenReady.
func newPluginHost(st *appState, dir, resources string,
	saved state.PluginState) (*pluginHost, error) {
	h := &pluginHost{
		st:            st,
		dir:           dir,
		resources:     resources,
		userDir:       filepath.Join(dir, "plugins"),
		root:          st.root,
		assets:        map[string][]byte{},
		instances:     map[string]*pluginInstance{},
		byPage:        map[*mygo.Page]*pluginInstance{},
		byHost:        map[string]*pluginInstance{},
		reports:       map[string]plugin.Report{},
		libraryIDs:    map[string]bool{},
		contracts:     map[string]plugin.ContractResult{},
		stats:         map[string]plugin.PluginStats{},
		details:       map[string]plugin.PluginDetail{},
		detailLoading: map[string]bool{},
		saved:         saved,
		ui: pluginUI{
			tab:           "browse",
			supportedOnly: true,
			browseLimit:   40,
			lastSupported: true,
			busy:          map[string]bool{},
			open:          map[string]bool{},
		},
	}
	for _, name := range []string{"host.html", "shim.js", "runtime.js"} {
		data, err := os.ReadFile(filepath.Join(h.resources, "obsidian-shim", name))
		if err != nil {
			slog.Warn("obsidian shim resource missing", "file", name, "err", err)
			continue
		}
		h.assets[name] = data
	}
	if err := mygo.Protocol.HandleFunc(pluginScheme, h.serve); err != nil {
		return nil, fmt.Errorf("register plugin scheme: %w", err)
	}
	h.refreshCompatible()
	h.refreshLibrary()
	h.refresh()
	h.updateState()
	return h, nil
}

// refresh rereads plugin packages and the compatibility list.
func (h *pluginHost) refresh() {
	reports, issues := h.discover()
	contracts := h.scanContracts(reports)
	h.mu.Lock()
	h.reports = reports
	h.contracts = contracts
	installed := make([]plugin.Report, 0, len(reports)+len(issues))
	for _, one := range reports {
		installed = append(installed, one)
	}
	installed = append(installed, issues...)
	sort.Slice(installed, func(i, j int) bool {
		return strings.ToLower(installed[i].Name) <
			strings.ToLower(installed[j].Name)
	})
	h.ui.installed = installed
	h.mu.Unlock()
}

// updateState rebuilds the snapshot on the main thread. It is called from
// stateChanged, which the view callbacks and the protocol bridge reach
// through mygo.RunOnMain.
func (h *pluginHost) updateState() {
	h.mu.RLock()
	root := h.root
	h.mu.RUnlock()
	snap := pluginStatePayload{
		APIVersion: plugin.APIProfile,
		Theme:      h.st.config.Basic.Theme,
		Language:   h.st.config.Basic.Language,
		Platform:   runtime.GOOS,
		Dark:       h.st.config.Basic.Appearance == config.AppearanceDark,
		Files:      []pluginFile{},
	}
	if root != "" {
		snap.VaultName = filepath.Base(root)
	}
	if h.st.idx != nil {
		paths := h.st.idx.Paths()
		dirs := map[string]bool{}
		for _, one := range paths {
			rel := h.relativePath(one)
			if rel == "" {
				continue
			}
			name := pathpkg.Base(rel)
			dot := strings.LastIndex(name, ".")
			snap.Files = append(snap.Files, pluginFile{
				Path: rel, Name: name,
				Extension: func() string {
					if dot > 0 {
						return name[dot+1:]
					}
					return ""
				}(),
				Basename: func() string {
					if dot > 0 {
						return name[:dot]
					}
					return name
				}(),
			})
			for dir := pathpkg.Dir(rel); dir != "." && dir != "/" && dir != ""; dir = pathpkg.Dir(dir) {
				dirs[dir] = true
			}
		}
		for dir := range dirs {
			snap.Files = append(snap.Files, pluginFile{
				Path: dir, Name: pathpkg.Base(dir), Folder: true,
			})
		}
		sort.Slice(snap.Files, func(i, j int) bool {
			return snap.Files[i].Path < snap.Files[j].Path
		})
	}
	if h.st.ws != nil && h.st.ws.Tabs != nil {
		if n := h.st.ws.Tabs.ActiveNote(); n != nil && n.Editor != nil {
			text := n.Editor.Buffer.String()
			sel := n.Editor.Selection()
			pos := n.Editor.Buffer.Pos(sel.End())
			snap.HasActiveNote = !n.Preview
			snap.ActivePath = h.relativePath(n.Path)
			snap.EditorText = text
			snap.SelectionStart = sel.Start()
			snap.SelectionEnd = sel.End()
			snap.CursorLine = pos.Line
			snap.CursorCh = pos.Col
		}
	}
	h.mu.Lock()
	h.revision++
	snap.Revision = h.revision
	h.snapshot = snap
	pages := make([]*mygo.Page, 0, len(h.instances))
	for _, inst := range h.instances {
		if inst.page != nil {
			pages = append(pages, inst.page.Page())
		}
	}
	h.mu.Unlock()
	for _, page := range pages {
		if err := pluginEvents.EmitPage(page, pluginEvent{
			Type: "stateChanged", Revision: snap.Revision,
		}); err != nil {
			slog.Debug("plugin state event", "err", err)
		}
	}
}

// stateChanged is the main-thread form of updateState plus a named event.
func (h *pluginHost) stateChanged(name string) {
	h.updateState()
	if name == "" || name == "stateChanged" {
		return
	}
	h.mu.RLock()
	pages := make([]*mygo.Page, 0, len(h.instances))
	for _, inst := range h.instances {
		if inst.page != nil {
			pages = append(pages, inst.page.Page())
		}
	}
	h.mu.RUnlock()
	for _, page := range pages {
		_ = pluginEvents.EmitPage(page, pluginEvent{
			Type: "event", Name: name,
		})
	}
}

// notifyState reaches the main thread from a protocol goroutine.
func (h *pluginHost) notifyState(name string) {
	mygo.RunOnMain(func() { h.stateChanged(name) })
}

// refreshCommands publishes the registered plugin commands to the native
// command palette. Built-in core plugins contribute to the same lists, so
// the merge lives in publishCoreContributions, which this calls.
func (h *pluginHost) refreshCommands() {
	h.publishCoreContributions()
}

func viewInList(list []view.PluginView, id string) bool {
	for _, one := range list {
		if one.ID == id {
			return true
		}
	}
	return false
}

// runAction delivers a ribbon click to the plugin page that registered it.
func (h *pluginHost) runAction(id string) {
	h.mu.RLock()
	var inst *pluginInstance
	for _, one := range h.instances {
		if _, ok := one.ribbon[id]; ok {
			inst = one
			break
		}
	}
	if inst == nil {
		h.mu.RUnlock()
		return
	}
	page, ready := inst.page, inst.ready
	h.mu.RUnlock()
	if !ready || page == nil {
		h.setStatus(locale.PluginStatus(locale.PluginCommandNotReady()))
		return
	}
	if err := pluginEvents.EmitPage(page.Page(), pluginEvent{
		Type: "action", ActionID: id,
	}); err != nil {
		slog.Warn("plugin ribbon event", "id", id, "err", err)
	}
}

// runCommand delivers a palette choice to the plugin page that owns it.
func (h *pluginHost) runCommand(id string) {
	h.mu.RLock()
	var inst *pluginInstance
	for _, one := range h.instances {
		if _, ok := one.commands[id]; ok {
			inst = one
			break
		}
	}
	if inst == nil {
		h.mu.RUnlock()
		return
	}
	page, ready := inst.page, inst.ready
	h.mu.RUnlock()
	if !ready || page == nil {
		h.setStatus(locale.PluginStatus(locale.PluginCommandNotReady()))
		return
	}
	if err := pluginEvents.EmitPage(page.Page(), pluginEvent{
		Type: "command", CommandID: id,
	}); err != nil {
		slog.Warn("plugin command event", "id", id, "err", err)
	}
}

// startEnabled loads the plugins the user left enabled. It runs after the
// main window exists, so NewWebView can attach to native UI.
func (h *pluginHost) startEnabled() {
	h.mu.RLock()
	ids := make([]string, 0, len(h.saved.Enabled))
	for id := range h.saved.Enabled {
		ids = append(ids, id)
	}
	h.mu.RUnlock()
	sort.Strings(ids)
	for _, id := range ids {
		h.enable(id, false)
	}
}

// enable creates one isolated WebView for a plugin. The page is untrusted;
// its identity comes from CallerPage, not from anything in the request.
func (h *pluginHost) enable(id string, save bool) {
	// A library plugin must pass the capability contract before any
	// WebView is made for it. External (L2) plugins are not held to it.
	if h.contractRefused(id) {
		h.setStatus(locale.PluginStatus(locale.PluginContractRefused()))
		return
	}
	if !mygo.WebViewAvailable() {
		h.setStatus(locale.PluginStatus(locale.PluginNoWebView()))
		return
	}
	h.mu.Lock()
	report, ok := h.reports[id]
	if !ok || !report.Supported {
		h.mu.Unlock()
		h.setStatus(locale.PluginStatus(locale.PluginInstallRefused()))
		return
	}
	if _, running := h.instances[id]; running {
		h.mu.Unlock()
		return
	}
	host, token := randomPluginID(), randomPluginID()
	m := h.manifestOf(report)
	m.Dir = virtualPluginDir(report.Source, id)
	inst := &pluginInstance{
		id: id, host: host, token: token, dir: report.Dir,
		source: report.Source, manifest: m, report: report,
		grants:   map[string]bool{},
		commands: map[string]view.PluginCommand{},
		ribbon:   map[string]view.PluginAction{},
		views:    map[string]view.PluginView{},
	}
	for _, one := range h.saved.GrantsOf(id) {
		inst.grants[one] = true
	}
	h.instances[id] = inst
	h.byHost[host] = inst
	if save {
		h.saved = h.saved.WithEnabled(id, true)
	}
	saved := h.saved
	h.mu.Unlock()
	if save {
		if err := state.SavePlugins(h.dir, saved); err != nil {
			slog.Warn("plugin state not written", "err", err)
		}
	}
	h.setStatus(locale.PluginStatus(locale.PluginEnabled()))
	mygo.RunOnMain(func() { h.createWebView(inst) })
}

// createWebView makes the page and installs explicit deny handlers. It
// runs on the main thread, as all window and WebView methods must.
func (h *pluginHost) createWebView(inst *pluginInstance) {
	if h.st.win == nil {
		h.failInstance(inst, "main window is not open")
		return
	}
	page, err := h.st.win.NewWebView(mygo.WebViewOptions{
		URL: "about:blank",
		Page: mygo.PageOptions{
			Untrusted:            true,
			Session:              mygo.NewEphemeralSession(),
			BackgroundThrottling: mygo.ThrottlingOff,
		},
	})
	if err != nil {
		h.failInstance(inst, err.Error())
		return
	}
	p := page.Page()
	p.SetWindowOpenHandler(func(mygo.WindowOpenRequest) *mygo.WindowOptions {
		return nil
	})
	p.OnWillNavigate(func(e *mygo.NavigateEvent) {
		if !h.allowedURL(e.URL, inst.host) {
			e.PreventDefault()
		}
	})
	p.SetPermissionHandler(func(mygo.PermissionRequest) bool { return false })
	p.OnWillDownload(func(e *mygo.DownloadEvent) { e.PreventDefault() })
	h.mu.Lock()
	inst.page = page
	h.byPage[p] = inst
	inst.timer = time.AfterFunc(12*time.Second, func() {
		h.mu.Lock()
		ready := inst.ready
		h.mu.Unlock()
		if !ready {
			h.failInstance(inst, "plugin did not become ready")
		}
	})
	h.mu.Unlock()
	if err := p.LoadURL(pluginScheme + "://" + inst.host + "/"); err != nil {
		h.failInstance(inst, err.Error())
	}
}

// disable asks the page to unload. The WebView is destroyed only after an
// acknowledgement or a short timeout, so Go's native registrations do not
// outlive the plugin.
func (h *pluginHost) disable(id string, save bool) {
	h.mu.Lock()
	inst := h.instances[id]
	if inst != nil {
		inst.ready = false
		inst.unloadID = randomPluginID()
	}
	if save {
		h.saved = h.saved.WithEnabled(id, false)
	}
	saved := h.saved
	h.mu.Unlock()
	if save {
		if err := state.SavePlugins(h.dir, saved); err != nil {
			slog.Warn("plugin state not written", "err", err)
		}
	}
	h.refreshCommands()
	h.setStatus(locale.PluginStatus(locale.PluginDisabled()))
	if inst == nil || inst.page == nil {
		h.finishUnload(inst, "")
		return
	}
	if err := pluginEvents.EmitPage(inst.page.Page(), pluginEvent{
		Type: "unload", RequestID: inst.unloadID,
	}); err != nil {
		slog.Warn("plugin unload event", "id", id, "err", err)
	}
	time.AfterFunc(2*time.Second, func() { h.finishUnload(inst, inst.unloadID) })
}

// reload is a disable followed by a short wait for the unload ack.
func (h *pluginHost) reload(id string) {
	h.disable(id, false)
	time.AfterFunc(300*time.Millisecond, func() { h.enable(id, false) })
}

// finishUnload removes an instance and destroys its WebView.
func (h *pluginHost) finishUnload(inst *pluginInstance, requestID string) {
	if inst == nil {
		return
	}
	h.mu.Lock()
	if requestID != "" && inst.unloadID != requestID {
		h.mu.Unlock()
		return
	}
	if _, ok := h.instances[inst.id]; !ok {
		h.mu.Unlock()
		return
	}
	if inst.timer != nil {
		inst.timer.Stop()
	}
	page := inst.page
	delete(h.instances, inst.id)
	delete(h.byHost, inst.host)
	if page != nil {
		delete(h.byPage, page.Page())
	}
	h.mu.Unlock()
	mygo.RunOnMain(func() {
		if page != nil {
			page.Destroy()
		}
	})
	h.refreshCommands()
}

// setEnabled is what the plugin manager calls for its toggle.
func (h *pluginHost) setEnabled(id string, on bool) {
	if on {
		h.enable(id, true)
	} else {
		h.disable(id, true)
	}
}

// setGrant changes one capability and saves the choice.
func (h *pluginHost) setGrant(id string, capability string, on bool) {
	h.mu.Lock()
	grants := h.saved.GrantsOf(id)
	kept := make([]string, 0, len(grants)+1)
	for _, one := range grants {
		if one != capability {
			kept = append(kept, one)
		}
	}
	if on {
		kept = append(kept, capability)
	}
	h.saved = h.saved.WithGrants(id, kept)
	if inst := h.instances[id]; inst != nil {
		for key := range inst.grants {
			delete(inst.grants, key)
		}
		for _, one := range kept {
			inst.grants[one] = true
		}
	}
	saved := h.saved
	h.mu.Unlock()
	if err := state.SavePlugins(h.dir, saved); err != nil {
		slog.Warn("plugin grants not written", "err", err)
	}
}

// granted reports whether a plugin may use a capability. Core lifecycle
// APIs are always available to an enabled plugin.
func (h *pluginHost) granted(inst *pluginInstance, capability string) bool {
	switch capability {
	case string(plugin.CapabilityData), string(plugin.CapabilityCommands),
		string(plugin.CapabilityNotice):
		return true
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return inst.grants[capability]
}

// failInstance marks a plugin as failed and removes its page.
func (h *pluginHost) failInstance(inst *pluginInstance, reason string) {
	h.mu.Lock()
	inst.failed = true
	inst.ready = false
	if inst.timer != nil {
		inst.timer.Stop()
	}
	page := inst.page
	if page != nil {
		delete(h.byPage, page.Page())
	}
	inst.page = nil
	delete(h.instances, inst.id)
	delete(h.byHost, inst.host)
	h.mu.Unlock()
	slog.Warn("plugin failed", "id", inst.id, "reason", reason)
	h.setStatus(locale.PluginStatus(reason))
	h.refreshCommands()
	mygo.RunOnMain(func() {
		if page != nil {
			page.Destroy()
		}
	})
}

// serve is the one HTTP entry point of the plugin scheme.
func (h *pluginHost) serve(w http.ResponseWriter, r *http.Request) {
	page := mygo.CallerPage(r.Context())
	h.mu.RLock()
	inst := h.byPage[page]
	h.mu.RUnlock()
	if inst == nil || r.URL.Host != inst.host {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Security-Policy", pluginCSP)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	switch {
	case r.URL.Path == "/" && r.Method == http.MethodGet:
		h.serveAsset(w, "host.html", "text/html; charset=utf-8")
	case r.URL.Path == "/config.js" && r.Method == http.MethodGet:
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		_, _ = w.Write(h.configJS(inst))
	case r.URL.Path == "/shim.js" && r.Method == http.MethodGet:
		h.serveAsset(w, "shim.js", "application/javascript; charset=utf-8")
	case r.URL.Path == "/runtime.js" && r.Method == http.MethodGet:
		h.serveAsset(w, "runtime.js", "application/javascript; charset=utf-8")
	case r.URL.Path == "/plugin/main.js" && r.Method == http.MethodGet:
		h.servePluginFile(w, inst, "main.js", "application/javascript; charset=utf-8")
	case r.URL.Path == "/plugin/styles.css" && r.Method == http.MethodGet:
		h.servePluginCSS(w, inst)
	case strings.HasPrefix(r.URL.Path, "/plugin/assets/") && r.Method == http.MethodGet:
		h.servePluginAsset(w, inst, strings.TrimPrefix(r.URL.Path,
			"/plugin/assets/"))
	case strings.HasPrefix(r.URL.Path, "/vault/") && r.Method == http.MethodGet:
		h.serveVaultFile(w, inst, strings.TrimPrefix(r.URL.Path, "/vault/"))
	case r.URL.Path == "/state" && r.Method == http.MethodGet:
		h.serveState(w)
	case r.URL.Path == "/rpc" && r.Method == http.MethodPost:
		h.serveRPC(w, r, inst)
	default:
		http.NotFound(w, r)
	}
}

func (h *pluginHost) serveAsset(w http.ResponseWriter, name, contentType string) {
	h.mu.RLock()
	data := h.assets[name]
	h.mu.RUnlock()
	if len(data) == 0 {
		http.Error(w, "missing shim resource", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write(data)
}

func (h *pluginHost) configJS(inst *pluginInstance) []byte {
	h.mu.RLock()
	granted := h.saved.GrantsOf(inst.id)
	h.mu.RUnlock()
	grants := make(map[string]bool, len(granted))
	for _, one := range granted {
		grants[one] = true
	}
	cfg := map[string]any{
		"pluginID":   inst.id,
		"pluginName": inst.manifest.Name,
		"host":       inst.host,
		"token":      inst.token,
		"apiVersion": plugin.APIProfile,
		"manifest":   inst.manifest,
		"grants":     grants,
	}
	data, _ := json.Marshal(cfg)
	return []byte("window.__XINKGO__ = " + string(data) + ";\n" +
		"document.body.dataset.pluginId = " +
		jsonString(inst.id) + ";\n")
}

func jsonString(text string) string {
	data, _ := json.Marshal(text)
	return string(data)
}

func (h *pluginHost) serveState(w http.ResponseWriter) {
	h.mu.RLock()
	snap := h.snapshot
	h.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(snap)
}

func (h *pluginHost) servePluginFile(w http.ResponseWriter,
	inst *pluginInstance, name, contentType string) {
	path := filepath.Join(inst.dir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		http.NotFound(w, nil)
		return
	}
	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write(data)
}

func (h *pluginHost) servePluginCSS(w http.ResponseWriter, inst *pluginInstance) {
	path := filepath.Join(inst.dir, "styles.css")
	data, err := os.ReadFile(path)
	if err != nil {
		http.NotFound(w, nil)
		return
	}
	text := cssImportPattern.ReplaceAllString(string(data), "")
	text = cssRemoteURL.ReplaceAllString(text, "none")
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	_, _ = w.Write([]byte(text))
}

func (h *pluginHost) servePluginAsset(w http.ResponseWriter,
	inst *pluginInstance, name string) {
	path, err := h.pluginFilePath(inst, name)
	if err != nil {
		http.NotFound(w, nil)
		return
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > 4<<20 {
		http.NotFound(w, nil)
		return
	}
	w.Header().Set("Content-Type", http.DetectContentType(data))
	_, _ = w.Write(data)
}

func (h *pluginHost) serveVaultFile(w http.ResponseWriter,
	inst *pluginInstance, raw string) {
	if !h.granted(inst, string(plugin.CapabilityVaultRead)) {
		http.Error(w, "vault.read is not granted", http.StatusForbidden)
		return
	}
	path, err := h.vaultPath(raw)
	if err != nil {
		http.NotFound(w, nil)
		return
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > 8<<20 {
		http.NotFound(w, nil)
		return
	}
	w.Header().Set("Content-Type", http.DetectContentType(data))
	_, _ = w.Write(data)
}

func (h *pluginHost) allowedURL(raw, host string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Scheme, pluginScheme) &&
		strings.EqualFold(u.Host, host)
}

func (h *pluginHost) pluginFilePath(inst *pluginInstance, name string) (string, error) {
	name = strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
	if name == "" || strings.Contains(name, "\x00") {
		return "", errors.New("bad asset path")
	}
	clean := pathpkg.Clean("/" + name)
	clean = strings.TrimPrefix(clean, "/")
	if clean == "" || clean == "." || clean == ".." ||
		strings.HasPrefix(clean, "../") {
		return "", errors.New("bad asset path")
	}
	full := filepath.Join(inst.dir, filepath.FromSlash(clean))
	rel, err := filepath.Rel(inst.dir, full)
	if err != nil || rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("asset path escapes plugin folder")
	}
	return full, nil
}

// vaultPath maps an Obsidian vault-relative path to a real path inside
// the open vault, refusing traversal and symlink escapes.
func (h *pluginHost) vaultPath(raw string) (string, error) {
	h.mu.RLock()
	root := h.root
	h.mu.RUnlock()
	raw = strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if raw == "" || root == "" || strings.Contains(raw, "\x00") {
		return "", errors.New("bad vault path")
	}
	if strings.HasPrefix(raw, "/") || filepath.IsAbs(raw) ||
		(len(raw) > 2 && raw[1] == ':') {
		return "", errors.New("absolute vault path")
	}
	clean := pathpkg.Clean(raw)
	if clean == "." || clean == ".." ||
		strings.HasPrefix(clean, "../") {
		return "", errors.New("vault path escapes root")
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		resolvedRoot = filepath.Clean(root)
	}
	full := filepath.Join(resolvedRoot, filepath.FromSlash(clean))
	resolved := full
	if _, err := os.Lstat(full); err == nil {
		if real, err := filepath.EvalSymlinks(full); err == nil {
			resolved = real
		}
	} else {
		parent := filepath.Dir(full)
		for {
			if _, err := os.Lstat(parent); err == nil {
				if real, err := filepath.EvalSymlinks(parent); err == nil {
					resolved = filepath.Join(real, filepath.Base(full))
				}
				break
			}
			next := filepath.Dir(parent)
			if next == parent {
				break
			}
			parent = next
		}
	}
	rel, err := filepath.Rel(resolvedRoot, resolved)
	if err != nil || rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("vault path escapes root")
	}
	return resolved, nil
}

func (h *pluginHost) relativePath(abs string) string {
	if abs == "" {
		return ""
	}
	h.mu.RLock()
	root := h.root
	h.mu.RUnlock()
	if root != "" && filepath.IsAbs(abs) {
		if rel, err := filepath.Rel(root, abs); err == nil {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(abs)
}

// setRoot is called on the main thread when the open vault changes.
// Protocol handlers read the cached root instead of appState fields.
func (h *pluginHost) setRoot(root string) {
	h.mu.Lock()
	h.root = root
	h.mu.Unlock()
}

func (h *pluginHost) manifestOf(report plugin.Report) plugin.Manifest {
	data, err := os.ReadFile(filepath.Join(report.Dir, "manifest.json"))
	if err != nil {
		return plugin.Manifest{ID: report.ID, Name: report.Name,
			Version: report.Version, Description: report.Description,
			Author: report.Author}
	}
	m, err := plugin.ParseManifest(data)
	if err != nil {
		return plugin.Manifest{ID: report.ID, Name: report.Name,
			Version: report.Version, Description: report.Description,
			Author: report.Author}
	}
	return m
}

func virtualPluginDir(source, id string) string {
	switch source {
	case "compat":
		return ".obsidian/plugins/" + id
	case "vault":
		return ".xink/plugins/" + id
	default:
		return "plugins/" + id
	}
}

func randomPluginID() string {
	var data [12]byte
	if _, err := rand.Read(data[:]); err != nil {
		return fmt.Sprintf("p-%d", time.Now().UnixNano())
	}
	return "p-" + hex.EncodeToString(data[:])
}

func (h *pluginHost) setStatus(text string) {
	h.mu.Lock()
	h.ui.status = text
	h.mu.Unlock()
}

// pluginStateChanged is the app's small seam to the plugin host: view
// callbacks tell it what changed, and it reaches the main thread before
// reading the window's state.
func (s *appState) pluginStateChanged(name string) {
	if s.plugins != nil {
		s.plugins.notifyState(name)
	}
}

func (h *pluginHost) setBusy(id string, on bool) {
	h.mu.Lock()
	if h.ui.busy == nil {
		h.ui.busy = map[string]bool{}
	}
	if on {
		h.ui.busy[id] = true
	} else {
		delete(h.ui.busy, id)
	}
	h.mu.Unlock()
}

func (h *pluginHost) isBusy(id string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.ui.busy[id]
}

func (h *pluginHost) enabled(id string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, running := h.instances[id]
	return running || h.saved.EnabledOf(id)
}

func (h *pluginHost) running(id string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	inst := h.instances[id]
	return inst != nil && inst.ready && !inst.failed
}

func (h *pluginHost) grantsOf(id string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.saved.GrantsOf(id)
}

func (h *pluginHost) grantOn(id string, capability string) bool {
	for _, one := range h.grantsOf(id) {
		if one == capability {
			return true
		}
	}
	return false
}
