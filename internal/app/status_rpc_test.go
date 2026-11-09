package app

import (
	"encoding/json"
	"testing"

	"github.com/helightxu/xinkgo/internal/config"
	"github.com/helightxu/xinkgo/internal/plugin"
	"github.com/helightxu/xinkgo/internal/view"
)

func statusRPCHost(t *testing.T) (*pluginHost, *pluginInstance) {
	t.Helper()
	h := testPluginHost(t)
	h.st = &appState{ws: view.NewWorkspace(config.Default())}
	inst := &pluginInstance{
		id:     "demo-basic",
		grants: map[string]bool{string(plugin.CapabilityUI): true},
	}
	return h, inst
}

func TestStatusRPC_keepsEachPluginItemIndependent(t *testing.T) {
	h, inst := statusRPCHost(t)
	if _, err := h.handleRPC(inst, "status.add", nil); err != nil {
		t.Fatalf("status.add: %v", err)
	}
	one, _ := json.Marshal(map[string]string{
		"text": "Demo Basic: 3 runs",
	})
	if _, err := h.handleRPC(inst, "status.set", one); err != nil {
		t.Fatalf("status.set: %v", err)
	}
	other := &pluginInstance{
		id:     "demo-full",
		grants: map[string]bool{string(plugin.CapabilityUI): true},
	}
	two, _ := json.Marshal(map[string]string{
		"text": "Demo Full: 5 runs",
	})
	if _, err := h.handleRPC(other, "status.set", two); err != nil {
		t.Fatalf("second status.set: %v", err)
	}
	items := h.st.ws.PluginStatuses
	if len(items) != 2 {
		t.Fatalf("plugin statuses = %+v", items)
	}
	if items[0].ID != "demo-basic" ||
		items[0].Text != "Demo Basic: 3 runs" ||
		items[1].ID != "demo-full" ||
		items[1].Text != "Demo Full: 5 runs" {
		t.Errorf("plugin items were merged or overwritten: %+v", items)
	}
}

func TestStatusRPC_requiresUIGrant(t *testing.T) {
	h, inst := statusRPCHost(t)
	inst.grants = map[string]bool{}
	if _, err := h.handleRPC(inst, "status.add", nil); err == nil {
		t.Error("status.add was allowed without ui")
	}
	raw, _ := json.Marshal(map[string]string{"text": "hello"})
	if _, err := h.handleRPC(inst, "status.set", raw); err == nil {
		t.Error("status.set was allowed without ui")
	}
	if len(h.st.ws.PluginStatuses) != 0 {
		t.Errorf("an unauthorised plugin left a status item: %+v",
			h.st.ws.PluginStatuses)
	}
}
