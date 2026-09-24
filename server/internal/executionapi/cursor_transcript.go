package executionapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const toolEvidenceSourceCursorTranscript = "cursor_transcript"

// FindCursorAgentTranscript locates Cursor's agent-transcript JSONL for a chat id.
// Transcripts live under ~/.cursor/projects/<slug>/agent-transcripts/<chatID>/<chatID>.jsonl
// and are the only place daemon-launched cursor-agent sessions record tool_use events
// (stdout session logs do not).
func FindCursorAgentTranscript(chatID, workingDir string) string {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	projectsRoot := filepath.Join(home, ".cursor", "projects")
	if slug := cursorProjectSlug(workingDir); slug != "" {
		candidate := filepath.Join(projectsRoot, slug, "agent-transcripts", chatID, chatID+".jsonl")
		if fileExists(candidate) {
			return candidate
		}
	}
	matches, err := filepath.Glob(filepath.Join(projectsRoot, "*", "agent-transcripts", chatID, chatID+".jsonl"))
	if err != nil || len(matches) == 0 {
		return ""
	}
	return matches[0]
}

// ReadCursorAgentTranscript returns transcript text when the chat id resolves.
func ReadCursorAgentTranscript(chatID, workingDir string) string {
	path := FindCursorAgentTranscript(chatID, workingDir)
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// ExtractToolCallsFromCursorTranscript parses Cursor agent-transcript JSONL for MCP/tool evidence.
func ExtractToolCallsFromCursorTranscript(transcript string, at time.Time) []ToolCallEvidence {
	if strings.TrimSpace(transcript) == "" {
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
			Source:   toolEvidenceSourceCursorTranscript,
			Status:   "observed",
			Summary:  truncateToolSummary(summary),
		}
	}

	for _, line := range strings.Split(transcript, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		var entry struct {
			Role    string `json:"role"`
			Message struct {
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.Role != "assistant" || len(entry.Message.Content) == 0 {
			continue
		}
		extractCursorContentTools(entry.Message.Content, record)
	}

	// Also accept regex/text patterns (CallMcpTool toolName=..., etc.).
	for _, call := range extractKnownToolsFromText(transcript, toolEvidenceSourceCursorTranscript, "observed", at) {
		record(call.Name, call.Summary)
	}

	out := make([]ToolCallEvidence, 0, len(found))
	for _, call := range found {
		out = append(out, call)
	}
	return compactToolCalls(out)
}

func extractCursorContentTools(raw json.RawMessage, record func(name, summary string)) {
	var blocks []map[string]any
	if err := json.Unmarshal(raw, &blocks); err != nil {
		// Single content object or string — fall through to text scan only.
		return
	}
	for _, block := range blocks {
		if strings.ToLower(stringFromAny(block["type"])) != "tool_use" {
			continue
		}
		name := stringFromAny(block["name"])
		input, _ := block["input"].(map[string]any)
		switch {
		case strings.EqualFold(name, "CallMcpTool") || strings.EqualFold(name, "call_mcp_tool"):
			toolName := firstNonEmpty(
				stringFromAny(input["toolName"]),
				stringFromAny(input["tool_name"]),
				stringFromAny(input["tool"]),
				stringFromAny(input["name"]),
			)
			record(toolName, "CallMcpTool "+toolName)
		case knownToolSet[NormalizeToolName(name)]:
			record(name, name)
		}
	}
}

func cursorProjectSlug(workingDir string) string {
	workingDir = strings.TrimSpace(workingDir)
	if workingDir == "" {
		return ""
	}
	clean := filepath.Clean(workingDir)
	if filepath.IsAbs(clean) {
		clean = strings.TrimPrefix(clean, string(filepath.Separator))
		// Windows drive paths are not expected on the daemon host; keep simple.
	}
	clean = strings.ReplaceAll(clean, string(filepath.Separator), "-")
	clean = strings.ReplaceAll(clean, ".", "-")
	return clean
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
