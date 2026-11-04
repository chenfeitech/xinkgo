package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/helightxu/xinkgo/internal/locale"
	"github.com/helightxu/xinkgo/internal/plugin"
	"github.com/helightxu/xinkgo/internal/view"
)

type pluginRPCRequest struct {
	ID     uint64          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	Token  string          `json:"token"`
}

type pluginRPCReply struct {
	ID     uint64 `json:"id"`
	OK     bool   `json:"ok"`
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

type pluginCommandParam struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Visible bool   `json:"visible"`
	Icon    string `json:"icon"`
}

type pluginVisibilityParam struct {
	Updates []struct {
		ID      string `json:"id"`
		Visible bool   `json:"visible"`
	} `json:"updates"`
}

type pluginTextParam struct {
	Text     string `json:"text"`
	Duration int    `json:"duration"`
}

type pluginDataParam struct {
	Data any `json:"data"`
}

type pluginVaultParam struct {
	Path string `json:"path"`
	To   string `json:"to"`
	Text string `json:"text"`
	Data string `json:"data"`
}

type pluginEditorParam struct {
	Text           string `json:"text"`
	SelectionStart int    `json:"selectionStart"`
	SelectionEnd   int    `json:"selectionEnd"`
}

type pluginNetworkParam struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    struct {
		Text   string `json:"text"`
		Base64 string `json:"base64"`
	} `json:"body"`
	Throw bool `json:"throw"`
}

func (h *pluginHost) serveRPC(w http.ResponseWriter, r *http.Request,
	inst *pluginInstance) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 256<<10))
	if err != nil {
		writePluginRPC(w, pluginRPCReply{OK: false, Error: err.Error()})
		return
	}
	var req pluginRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writePluginRPC(w, pluginRPCReply{OK: false, Error: err.Error()})
		return
	}
	if req.Token != inst.token {
		writePluginRPC(w, pluginRPCReply{ID: req.ID, OK: false,
			Error: "plugin token mismatch"})
		return
	}
	result, err := h.handleRPC(inst, req.Method, req.Params)
	reply := pluginRPCReply{ID: req.ID, OK: err == nil, Result: result}
	if err != nil {
		reply.Error = err.Error()
	}
	writePluginRPC(w, reply)
}

func writePluginRPC(w http.ResponseWriter, reply pluginRPCReply) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(reply)
}

func decodePluginParams(raw json.RawMessage, out any) error {
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}

func (h *pluginHost) handleRPC(inst *pluginInstance, method string,
	raw json.RawMessage) (any, error) {
	switch method {
	case "ready":
		h.mu.Lock()
		inst.ready = true
		inst.failed = false
		if inst.timer != nil {
			inst.timer.Stop()
		}
		h.mu.Unlock()
		h.refreshCommands()
		h.setStatus(locale.PluginStatus(locale.PluginEnabled()))
		return nil, nil
	case "error":
		var p pluginTextParam
		if err := decodePluginParams(raw, &p); err != nil {
			return nil, err
		}
		h.failInstance(inst, p.Text)
		return nil, nil
	case "log":
		var p struct {
			Level string `json:"level"`
			Text  string `json:"text"`
		}
		if err := decodePluginParams(raw, &p); err != nil {
			return nil, err
		}
		slog.Info("plugin console", "id", inst.id, "level", p.Level,
			"text", p.Text)
		return nil, nil
	case "notice", "status.set":
		if method == "status.set" &&
			!h.granted(inst, string(plugin.CapabilityUI)) {
			return nil, errors.New("ui is not granted")
		}
		var p pluginTextParam
		if err := decodePluginParams(raw, &p); err != nil {
			return nil, err
		}
		h.st.say(p.Text)
		return nil, nil
	case "status.add":
		if !h.granted(inst, string(plugin.CapabilityUI)) {
			return nil, errors.New("ui is not granted")
		}
		return nil, nil
	case "data.load":
		return h.pluginData(inst)
	case "data.save":
		var p pluginDataParam
		if err := decodePluginParams(raw, &p); err != nil {
			return nil, err
		}
		data, err := json.MarshalIndent(p.Data, "", "  ")
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(inst.dir, 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(inst.dir, "data.json"),
			append(data, '\n'), 0o644); err != nil {
			return nil, err
		}
		return nil, nil
	case "command.add":
		var p pluginCommandParam
		if err := decodePluginParams(raw, &p); err != nil {
			return nil, err
		}
		if p.ID == "" || p.Name == "" {
			return nil, errors.New("command id and name are required")
		}
		h.mu.Lock()
		inst.commands[p.ID] = view.PluginCommand{
			ID: p.ID, Name: p.Name, Visible: p.Visible,
		}
		h.mu.Unlock()
		h.refreshCommands()
		return nil, nil
	case "command.remove":
		var p struct {
			ID string `json:"id"`
		}
		if err := decodePluginParams(raw, &p); err != nil {
			return nil, err
		}
		h.mu.Lock()
		delete(inst.commands, p.ID)
		h.mu.Unlock()
		h.refreshCommands()
		return nil, nil
	case "command.visibility":
		var p pluginVisibilityParam
		if err := decodePluginParams(raw, &p); err != nil {
			return nil, err
		}
		h.mu.Lock()
		for _, one := range p.Updates {
			if command, ok := inst.commands[one.ID]; ok {
				command.Visible = one.Visible
				inst.commands[one.ID] = command
			}
		}
		h.mu.Unlock()
		h.refreshCommands()
		return nil, nil
	case "action.add":
		if !h.granted(inst, string(plugin.CapabilityUI)) {
			return nil, errors.New("ui is not granted")
		}
		var p struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Kind string `json:"kind"`
		}
		if err := decodePluginParams(raw, &p); err != nil {
			return nil, err
		}
		commandID := inst.id + ":action:" + p.ID
		h.mu.Lock()
		inst.commands[commandID] = view.PluginCommand{
			ID: commandID, Name: p.Name, Visible: true,
		}
		h.mu.Unlock()
		h.refreshCommands()
		return nil, nil
	case "editor.write":
		if !h.granted(inst, string(plugin.CapabilityEditorWrite)) {
			return nil, errors.New("editor.write is not granted")
		}
		var p pluginEditorParam
		if err := decodePluginParams(raw, &p); err != nil {
			return nil, err
		}
		if err := h.setEditor(p.Text, p.SelectionStart,
			p.SelectionEnd); err != nil {
			return nil, err
		}
		h.notifyState("editor-change")
		return nil, nil
	case "workspace.open":
		if !h.granted(inst, string(plugin.CapabilityVaultRead)) {
			return nil, errors.New("vault.read is not granted")
		}
		var p struct {
			Linktext string `json:"linktext"`
		}
		if err := decodePluginParams(raw, &p); err != nil {
			return nil, err
		}
		return h.openWorkspaceLink(p.Linktext)
	case "metadata.get":
		if !h.granted(inst, string(plugin.CapabilityMetadata)) {
			return nil, errors.New("metadata is not granted")
		}
		return map[string]any{}, nil
	case "network.request":
		if !h.granted(inst, string(plugin.CapabilityNetwork)) {
			return nil, errors.New("network is not granted")
		}
		var p pluginNetworkParam
		if err := decodePluginParams(raw, &p); err != nil {
			return nil, err
		}
		return h.networkRequest(p)
	case "unloadDone":
		var p struct {
			RequestID string `json:"requestID"`
		}
		if err := decodePluginParams(raw, &p); err != nil {
			return nil, err
		}
		h.finishUnload(inst, p.RequestID)
		return nil, nil
	case "vault.exists":
		return h.vaultExists(inst, raw)
	case "vault.stat":
		return h.vaultStat(inst, raw)
	case "vault.list":
		return h.vaultList(inst, raw)
	case "vault.read":
		return h.vaultRead(inst, raw)
	case "vault.readBinary":
		return h.vaultReadBinary(inst, raw)
	case "vault.write", "vault.create":
		return h.vaultWrite(inst, raw, method == "vault.create")
	case "vault.writeBinary":
		return h.vaultWriteBinary(inst, raw)
	case "vault.append":
		return h.vaultAppend(inst, raw)
	case "vault.createFolder":
		return h.vaultCreateFolder(inst, raw)
	case "vault.delete":
		return h.vaultDelete(inst, raw)
	case "vault.rename":
		return h.vaultRename(inst, raw)
	default:
		return nil, fmt.Errorf("unknown plugin method %q", method)
	}
}

func (h *pluginHost) pluginData(inst *pluginInstance) (any, error) {
	data, err := os.ReadFile(filepath.Join(inst.dir, "data.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func (h *pluginHost) setEditor(text string, start, end int) error {
	var result error
	mygo.RunOnMain(func() {
		if h.st.ws == nil || h.st.ws.Tabs == nil {
			result = errors.New("no window is open")
			return
		}
		n := h.st.ws.Tabs.ActiveNote()
		if n == nil || n.Editor == nil || n.Preview {
			result = errors.New("no editable note is active")
			return
		}
		size := len(text)
		if start < 0 || start > size {
			start = size
		}
		if end < 0 || end > size {
			end = size
		}
		n.Editor.Buffer.Set(text)
		n.Editor.Select(start, end)
	})
	return result
}

func (h *pluginHost) openWorkspaceLink(linktext string) (any, error) {
	link := strings.Split(strings.Split(linktext, "#")[0], "|")[0]
	path, err := h.vaultPath(link)
	if err != nil {
		return nil, err
	}
	var result error
	mygo.RunOnMain(func() {
		if h.st.ws == nil || h.st.ws.Tabs == nil {
			result = errors.New("no window is open")
			return
		}
		if _, err := h.st.ws.Tabs.Open(path); err != nil {
			result = err
			return
		}
		h.st.ws.Tree.Reveal(path)
	})
	if result != nil {
		return nil, result
	}
	h.notifyState("file-open")
	return nil, nil
}

func (h *pluginHost) vaultExists(inst *pluginInstance,
	raw json.RawMessage) (any, error) {
	if !h.granted(inst, string(plugin.CapabilityVaultRead)) {
		return nil, errors.New("vault.read is not granted")
	}
	var p pluginVaultParam
	if err := decodePluginParams(raw, &p); err != nil {
		return nil, err
	}
	path, err := h.vaultPath(p.Path)
	if err != nil {
		return false, nil
	}
	_, err = os.Stat(path)
	return err == nil, nil
}

func (h *pluginHost) vaultStat(inst *pluginInstance,
	raw json.RawMessage) (any, error) {
	if !h.granted(inst, string(plugin.CapabilityVaultRead)) {
		return nil, errors.New("vault.read is not granted")
	}
	var p pluginVaultParam
	if err := decodePluginParams(raw, &p); err != nil {
		return nil, err
	}
	path, err := h.vaultPath(p.Path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, nil
	}
	kind := "file"
	if info.IsDir() {
		kind = "folder"
	}
	return map[string]any{
		"ctime": info.ModTime().UnixMilli(),
		"mtime": info.ModTime().UnixMilli(),
		"size":  info.Size(),
		"type":  kind,
	}, nil
}

func (h *pluginHost) vaultList(inst *pluginInstance,
	raw json.RawMessage) (any, error) {
	if !h.granted(inst, string(plugin.CapabilityVaultRead)) {
		return nil, errors.New("vault.read is not granted")
	}
	var p pluginVaultParam
	if err := decodePluginParams(raw, &p); err != nil {
		return nil, err
	}
	dir, err := h.vaultPath(p.Path)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files, folders := []string{}, []string{}
	for _, one := range entries {
		if one.IsDir() {
			folders = append(folders, one.Name())
		} else {
			files = append(files, one.Name())
		}
	}
	return map[string]any{"files": files, "folders": folders}, nil
}

func (h *pluginHost) vaultRead(inst *pluginInstance,
	raw json.RawMessage) (any, error) {
	if !h.granted(inst, string(plugin.CapabilityVaultRead)) {
		return nil, errors.New("vault.read is not granted")
	}
	var p pluginVaultParam
	if err := decodePluginParams(raw, &p); err != nil {
		return nil, err
	}
	path, err := h.vaultPath(p.Path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

func (h *pluginHost) vaultReadBinary(inst *pluginInstance,
	raw json.RawMessage) (any, error) {
	if !h.granted(inst, string(plugin.CapabilityVaultRead)) {
		return nil, errors.New("vault.read is not granted")
	}
	var p pluginVaultParam
	if err := decodePluginParams(raw, &p); err != nil {
		return nil, err
	}
	path, err := h.vaultPath(p.Path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > 8<<20 {
		return nil, errors.New("binary file could not be read")
	}
	return map[string]any{"base64": base64.StdEncoding.EncodeToString(data)}, nil
}

func (h *pluginHost) vaultWrite(inst *pluginInstance,
	raw json.RawMessage, create bool) (any, error) {
	if !h.granted(inst, string(plugin.CapabilityVaultWrite)) {
		return nil, errors.New("vault.write is not granted")
	}
	var p pluginVaultParam
	if err := decodePluginParams(raw, &p); err != nil {
		return nil, err
	}
	path, err := h.vaultPath(p.Path)
	if err != nil {
		return nil, err
	}
	if create {
		if _, err := os.Stat(path); err == nil {
			return nil, errors.New("file already exists")
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, []byte(p.Text), 0o644); err != nil {
		return nil, err
	}
	h.notifyState("vault.modify")
	return map[string]any{"path": p.Path}, nil
}

func (h *pluginHost) vaultWriteBinary(inst *pluginInstance,
	raw json.RawMessage) (any, error) {
	if !h.granted(inst, string(plugin.CapabilityVaultWrite)) {
		return nil, errors.New("vault.write is not granted")
	}
	var p pluginVaultParam
	if err := decodePluginParams(raw, &p); err != nil {
		return nil, err
	}
	data, err := base64.StdEncoding.DecodeString(p.Data)
	if err != nil {
		return nil, err
	}
	path, err := h.vaultPath(p.Path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return nil, err
	}
	h.notifyState("vault.modify")
	return map[string]any{"path": p.Path}, nil
}

func (h *pluginHost) vaultAppend(inst *pluginInstance,
	raw json.RawMessage) (any, error) {
	if !h.granted(inst, string(plugin.CapabilityVaultWrite)) {
		return nil, errors.New("vault.write is not granted")
	}
	var p pluginVaultParam
	if err := decodePluginParams(raw, &p); err != nil {
		return nil, err
	}
	path, err := h.vaultPath(p.Path)
	if err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0o644)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if _, err := file.WriteString(p.Text); err != nil {
		return nil, err
	}
	h.notifyState("vault.modify")
	return nil, nil
}

func (h *pluginHost) vaultCreateFolder(inst *pluginInstance,
	raw json.RawMessage) (any, error) {
	if !h.granted(inst, string(plugin.CapabilityVaultWrite)) {
		return nil, errors.New("vault.write is not granted")
	}
	var p pluginVaultParam
	if err := decodePluginParams(raw, &p); err != nil {
		return nil, err
	}
	path, err := h.vaultPath(p.Path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return nil, err
	}
	h.notifyState("vault.create")
	return map[string]any{"path": p.Path}, nil
}

func (h *pluginHost) vaultDelete(inst *pluginInstance,
	raw json.RawMessage) (any, error) {
	if !h.granted(inst, string(plugin.CapabilityVaultWrite)) {
		return nil, errors.New("vault.write is not granted")
	}
	var p pluginVaultParam
	if err := decodePluginParams(raw, &p); err != nil {
		return nil, err
	}
	path, err := h.vaultPath(p.Path)
	if err != nil {
		return nil, err
	}
	if err := os.RemoveAll(path); err != nil {
		return nil, err
	}
	h.notifyState("vault.delete")
	return nil, nil
}

func (h *pluginHost) vaultRename(inst *pluginInstance,
	raw json.RawMessage) (any, error) {
	if !h.granted(inst, string(plugin.CapabilityVaultWrite)) {
		return nil, errors.New("vault.write is not granted")
	}
	var p pluginVaultParam
	if err := decodePluginParams(raw, &p); err != nil {
		return nil, err
	}
	from, err := h.vaultPath(p.Path)
	if err != nil {
		return nil, err
	}
	to, err := h.vaultPath(p.To)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return nil, err
	}
	if err := os.Rename(from, to); err != nil {
		return nil, err
	}
	h.notifyState("vault.rename")
	return nil, nil
}

func (h *pluginHost) networkRequest(p pluginNetworkParam) (any, error) {
	parsed, err := url.Parse(p.URL)
	if err != nil || parsed.Host == "" ||
		(parsed.Scheme != "https" && parsed.Scheme != "http") ||
		parsed.User != nil {
		return nil, errors.New("only http(s) URLs are allowed")
	}
	method := p.Method
	if method == "" {
		method = http.MethodGet
	}
	var body io.Reader
	if p.Body.Base64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(p.Body.Base64)
		if err != nil {
			return nil, err
		}
		body = strings.NewReader(string(decoded))
	} else if p.Body.Text != "" {
		body = strings.NewReader(p.Body.Text)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, p.URL, body)
	if err != nil {
		return nil, err
	}
	for key, value := range p.Headers {
		req.Header.Set(key, value)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if p.Throw && resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	headers := map[string]string{}
	for key := range resp.Header {
		headers[key] = resp.Header.Get(key)
	}
	return map[string]any{
		"status":  resp.StatusCode,
		"headers": headers,
		"text":    string(data),
		"base64":  base64.StdEncoding.EncodeToString(data),
	}, nil
}
