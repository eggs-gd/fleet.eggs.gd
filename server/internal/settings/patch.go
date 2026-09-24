package settings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type PatchRequest struct {
	ScanRoots *[]string             `json:"scanRoots"`
	Agents    map[string]AgentPatch `json:"agents"`
	Manager   *ManagerPatch         `json:"manager"`
}

type AgentPatch struct {
	Enabled             *bool   `json:"enabled"`
	Executable          *string `json:"executable"`
	RoutingInstructions *string `json:"routingInstructions"`
}

// ManagerPatch sets or clears the Manager session binding. Sending an empty
// agent (or an empty threadId) clears the binding entirely — a manager is
// either fully bound or not bound, there is no partial state.
type ManagerPatch struct {
	Agent    *string `json:"agent"`
	ThreadID *string `json:"threadId"`
}

type PatchResult struct {
	Snapshot Snapshot `json:"snapshot"`
	Warnings []string `json:"warnings,omitempty"`
}

func DecodePatch(body []byte) (PatchRequest, error) {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		body = []byte("{}")
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	var patch PatchRequest
	if err := dec.Decode(&patch); err != nil {
		return PatchRequest{}, fmt.Errorf("unknown or invalid settings field: %w", err)
	}
	for id := range patch.Agents {
		if !isKnownAgent(id) {
			return PatchRequest{}, fmt.Errorf("unknown agent %q", id)
		}
	}
	if patch.Manager != nil && patch.Manager.Agent != nil {
		id := strings.TrimSpace(*patch.Manager.Agent)
		if id != "" && !isKnownAgent(id) {
			return PatchRequest{}, fmt.Errorf("unknown manager agent %q", id)
		}
	}
	return patch, nil
}

func MergeOverlay(base Overlay, patch PatchRequest) Overlay {
	out := base.clone()
	if out.Agents == nil {
		out.Agents = map[string]AgentOverlay{}
	}
	if patch.ScanRoots != nil {
		out.ScanRoots = normalizeScanRoots(*patch.ScanRoots)
	}
	for id, item := range patch.Agents {
		cur := out.Agents[id]
		if item.Enabled != nil {
			cur.Enabled = item.Enabled
		}
		if item.Executable != nil {
			cur.Executable = strings.TrimSpace(*item.Executable)
		}
		if item.RoutingInstructions != nil {
			cur.RoutingInstructions = strings.TrimSpace(*item.RoutingInstructions)
		}
		if agentEmpty(cur) {
			delete(out.Agents, id)
			continue
		}
		out.Agents[id] = cur
	}
	if patch.Manager != nil {
		cur := out.Manager
		if patch.Manager.Agent != nil {
			cur.Agent = strings.TrimSpace(*patch.Manager.Agent)
		}
		if patch.Manager.ThreadID != nil {
			cur.ThreadID = strings.TrimSpace(*patch.Manager.ThreadID)
		}
		switch {
		case cur.Agent == "" || cur.ThreadID == "":
			cur = ManagerOverlay{}
		case cur.Agent != base.Manager.Agent || cur.ThreadID != base.Manager.ThreadID:
			cur.BoundAt = time.Now().UTC().Format(time.RFC3339)
		default:
			cur.BoundAt = base.Manager.BoundAt
		}
		out.Manager = cur
	}
	return out
}

func ValidateOverlay(overlay Overlay) (warnings []string, err error) {
	for _, root := range normalizeScanRoots(overlay.ScanRoots) {
		if err := ValidateScanRoot(root); err != nil {
			return nil, err
		}
	}
	for _, id := range knownAgentIDs {
		agent := overlay.Agent(id)
		if strings.TrimSpace(agent.Executable) == "" {
			continue
		}
		warning, err := ValidateExecutable(agent.Executable)
		if err != nil {
			return nil, fmt.Errorf("%s executable: %w", id, err)
		}
		if warning != "" {
			warnings = append(warnings, id+": "+warning)
		}
	}
	return warnings, nil
}
