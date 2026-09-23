package executionapi

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

var (
	// Matches Cursor/Claude-style MCP invocations mentioning a known tool.
	callMcpToolPattern = regexp.MustCompile(`(?i)(?:CallMcpTool|call_mcp_tool|mcp_tool_call)[^a-zA-Z0-9_-]{0,40}(?:toolName|tool_name|tool|name)\s*[:=]\s*["'\x60]?([a-zA-Z0-9_-]+)`)
	// Matches Anthropic-style tool_use name=... or "name":"...".
	toolUseNamePattern = regexp.MustCompile(`(?i)(?:tool_use|mcpToolCall|mcp_tool_call)[^\\n]{0,120}?(?:\\"|")?name(?:\\"|")?\s*[:=]\s*["'\x60]?([a-zA-Z0-9_-]+)`)
	// Matches JSON "tool":"go_diagnostics" / "toolName":"..." near MCP context.
	jsonToolFieldPattern = regexp.MustCompile(`(?i)"(?:tool|toolName|tool_name|name)"\s*:\s*"([a-zA-Z0-9_-]+)"`)
)

// knownToolSet is the closed set we treat as AGENTS.md-relevant evidence.
var knownToolSet = map[string]bool{
	ToolGoDiagnostics:    true,
	ToolGoVulncheck:      true,
	ToolSvelteAutofixer:  true,
	ToolListSections:     true,
	ToolGetDocumentation: true,
	ToolFindPatterns:     true,
	ToolFindSimilarCode:  true,
	"get_pattern_details": true,
	"search_patterns":     true,
	"go_references":       true,
	"go_symbol_references": true,
	"go_package_api":      true,
}

// ExtractToolCallsFromCodexParams extracts tool evidence from one Codex JSON-RPC params blob.
func ExtractToolCallsFromCodexParams(method string, raw json.RawMessage, at time.Time) []ToolCallEvidence {
	if len(raw) == 0 {
		return nil
	}
	method = strings.TrimSpace(method)
	status := "observed"
	switch method {
	case "item/started":
		status = "started"
	case "item/completed":
		status = "completed"
	}
	var payload struct {
		Item json.RawMessage `json:"item"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil || len(payload.Item) == 0 {
		return extractKnownToolsFromText(string(raw), toolEvidenceSourceRPC, status, at)
	}
	calls := extractToolCallsFromItemJSON(payload.Item, status, at)
	if len(calls) > 0 {
		return calls
	}
	return extractKnownToolsFromText(string(payload.Item), toolEvidenceSourceRPC, status, at)
}

// ExtractToolCallsFromSessionLog scans a provider session transcript/log for tool use.
// It deliberately ignores WorkerResult JSON payloads (self-report) as sole evidence.
func ExtractToolCallsFromSessionLog(logText string, at time.Time) []ToolCallEvidence {
	if strings.TrimSpace(logText) == "" {
		return nil
	}
	// Prefer structured Codex RPC lines when present.
	var calls []ToolCallEvidence
	for _, line := range strings.Split(logText, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		payload := line
		if idx := strings.Index(line, "{"); idx >= 0 {
			payload = line[idx:]
		}
		var msg struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal([]byte(payload), &msg); err == nil && strings.HasPrefix(msg.Method, "item/") {
			calls = append(calls, ExtractToolCallsFromCodexParams(msg.Method, msg.Params, at)...)
			continue
		}
	}
	calls = append(calls, extractKnownToolsFromText(logText, toolEvidenceSourceLog, "observed", at)...)
	return compactToolCalls(calls)
}

func extractToolCallsFromItemJSON(raw json.RawMessage, status string, at time.Time) []ToolCallEvidence {
	var item map[string]any
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil
	}
	itemType := strings.ToLower(stringFromAny(item["type"]))
	name := firstNonEmpty(
		stringFromAny(item["tool"]),
		stringFromAny(item["toolName"]),
		stringFromAny(item["tool_name"]),
		stringFromAny(item["name"]),
	)
	server := firstNonEmpty(stringFromAny(item["server"]), stringFromAny(item["mcpServer"]), stringFromAny(item["mcp_server"]))
	summary := firstNonEmpty(
		truncateToolSummary(stringFromAny(item["result"])),
		truncateToolSummary(stringFromAny(item["output"])),
		truncateToolSummary(stringFromAny(item["command"])),
		truncateToolSummary(server),
	)

	switch {
	case name != "" && (strings.Contains(itemType, "mcp") || strings.Contains(itemType, "tool") || knownToolSet[NormalizeToolName(name)]):
		return []ToolCallEvidence{{
			Name:     NormalizeToolName(name),
			CalledAt: formatToolEvidenceTime(at),
			Source:   toolEvidenceSourceRPC,
			Status:   status,
			Summary:  summary,
		}}
	case strings.Contains(itemType, "mcp") || itemType == "function_call" || itemType == "functioncall":
		if name == "" {
			name = stringFromAny(item["function"])
		}
		if name == "" {
			return nil
		}
		return []ToolCallEvidence{{
			Name:     NormalizeToolName(name),
			CalledAt: formatToolEvidenceTime(at),
			Source:   toolEvidenceSourceRPC,
			Status:   status,
			Summary:  summary,
		}}
	default:
		return nil
	}
}

func extractKnownToolsFromText(text string, source, status string, at time.Time) []ToolCallEvidence {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	// Skip pure worker-result self-report blobs: they can mention tool names in prose.
	if looksLikeWorkerResultOnly(text) {
		return nil
	}

	found := map[string]ToolCallEvidence{}
	record := func(rawName, summary string) {
		name := NormalizeToolName(rawName)
		if name == "" || !knownToolSet[name] {
			return
		}
		if _, ok := found[name]; ok {
			return
		}
		found[name] = ToolCallEvidence{
			Name:     name,
			CalledAt: formatToolEvidenceTime(at),
			Source:   source,
			Status:   status,
			Summary:  truncateToolSummary(summary),
		}
	}

	for _, match := range callMcpToolPattern.FindAllStringSubmatch(text, -1) {
		if len(match) > 1 {
			record(match[1], match[0])
		}
	}
	for _, match := range toolUseNamePattern.FindAllStringSubmatch(text, -1) {
		if len(match) > 1 {
			record(match[1], match[0])
		}
	}
	for _, match := range jsonToolFieldPattern.FindAllStringSubmatch(text, -1) {
		if len(match) > 1 && knownToolSet[NormalizeToolName(match[1])] {
			record(match[1], match[0])
		}
	}

	out := make([]ToolCallEvidence, 0, len(found))
	for _, call := range found {
		out = append(out, call)
	}
	return compactToolCalls(out)
}

func looksLikeWorkerResultOnly(text string) bool {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "{") || !strings.Contains(trimmed, `"outcome"`) {
		return false
	}
	// If the blob is small and looks like the CORE-97 payload, ignore it.
	if len(trimmed) < 2000 && !strings.Contains(trimmed, "CallMcpTool") && !strings.Contains(trimmed, "tool_use") && !strings.Contains(trimmed, "mcpToolCall") {
		var probe map[string]any
		if err := json.Unmarshal([]byte(trimmed), &probe); err == nil {
			if _, ok := probe["outcome"]; ok {
				return true
			}
		}
	}
	return false
}

func stringFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	case float64:
		return strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(jsonNumber(typed), ".0"), ".00"))
	default:
		return ""
	}
}

func jsonNumber(value float64) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(raw)
}
