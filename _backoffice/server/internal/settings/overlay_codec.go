package settings

import (
	"fmt"
	"strconv"
	"strings"
)

func EncodeOverlay(overlay Overlay) []byte {
	var b strings.Builder
	b.WriteString("# Machine-local Core overlay. Do not commit.\n")
	if strings.TrimSpace(overlay.ScanRoot) != "" {
		fmt.Fprintf(&b, "scanRoot: %s\n", yamlQuote(overlay.ScanRoot))
	}
	wroteAgents := false
	for _, id := range knownAgentIDs {
		agent, ok := overlay.Agents[id]
		if !ok || agentEmpty(agent) {
			continue
		}
		if !wroteAgents {
			b.WriteString("agents:\n")
			wroteAgents = true
		}
		fmt.Fprintf(&b, "  %s:\n", id)
		if agent.Enabled != nil {
			fmt.Fprintf(&b, "    enabled: %v\n", *agent.Enabled)
		}
		if strings.TrimSpace(agent.Executable) != "" {
			fmt.Fprintf(&b, "    executable: %s\n", yamlQuote(agent.Executable))
		}
		if strings.TrimSpace(agent.RoutingInstructions) != "" {
			b.WriteString("    routingInstructions: |\n")
			for _, line := range strings.Split(agent.RoutingInstructions, "\n") {
				fmt.Fprintf(&b, "      %s\n", line)
			}
		}
	}
	if !managerEmpty(overlay.Manager) {
		b.WriteString("manager:\n")
		if strings.TrimSpace(overlay.Manager.Agent) != "" {
			fmt.Fprintf(&b, "  agent: %s\n", yamlQuote(overlay.Manager.Agent))
		}
		if strings.TrimSpace(overlay.Manager.ThreadID) != "" {
			fmt.Fprintf(&b, "  threadId: %s\n", yamlQuote(overlay.Manager.ThreadID))
		}
		if strings.TrimSpace(overlay.Manager.BoundAt) != "" {
			fmt.Fprintf(&b, "  boundAt: %s\n", yamlQuote(overlay.Manager.BoundAt))
		}
	}
	return []byte(b.String())
}

func ParseOverlay(data []byte) (Overlay, error) {
	out := Overlay{Agents: map[string]AgentOverlay{}}
	lines := strings.Split(string(data), "\n")
	section := ""
	agentID := ""
	blockKey := ""
	blockIndent := 0
	var block []string

	flushBlock := func() {
		if blockKey == "" || agentID == "" {
			blockKey = ""
			block = nil
			return
		}
		cur := out.Agents[agentID]
		cur.RoutingInstructions = trimRouting(strings.Join(block, "\n"))
		out.Agents[agentID] = cur
		blockKey = ""
		block = nil
	}

	for i := 0; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r")
		if blockKey != "" {
			indent := leadingSpaces(line)
			if strings.TrimSpace(line) == "" {
				block = append(block, "")
				continue
			}
			if indent > blockIndent {
				prefix := blockIndent + 2
				if len(line) >= prefix {
					block = append(block, line[prefix:])
				} else {
					block = append(block, strings.TrimLeft(line, " "))
				}
				continue
			}
			flushBlock()
		}

		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		indent := leadingSpaces(line)
		key, raw, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			return Overlay{}, fmt.Errorf("line %d: expected key: value", i+1)
		}
		key = strings.TrimSpace(key)
		raw = strings.TrimSpace(raw)

		if indent == 0 {
			agentID = ""
			switch key {
			case "scanRoot":
				out.ScanRoot = unquoteYAML(raw)
				section = ""
			case "agents":
				section = "agents"
			case "manager":
				section = "manager"
			default:
				return Overlay{}, fmt.Errorf("unknown overlay key %q", key)
			}
			continue
		}
		if indent == 2 && section == "manager" {
			switch key {
			case "agent":
				out.Manager.Agent = unquoteYAML(raw)
			case "threadId":
				out.Manager.ThreadID = unquoteYAML(raw)
			case "boundAt":
				out.Manager.BoundAt = unquoteYAML(raw)
			default:
				return Overlay{}, fmt.Errorf("unknown overlay key manager.%s", key)
			}
			continue
		}
		if indent == 2 && section == "agents" {
			if !isKnownAgent(key) {
				return Overlay{}, fmt.Errorf("unknown agent %q", key)
			}
			agentID = key
			if _, ok := out.Agents[agentID]; !ok {
				out.Agents[agentID] = AgentOverlay{}
			}
			continue
		}
		if indent == 4 && agentID != "" {
			cur := out.Agents[agentID]
			switch key {
			case "enabled":
				value, err := parseBool(raw)
				if err != nil {
					return Overlay{}, fmt.Errorf("agents.%s.enabled: %w", agentID, err)
				}
				cur.Enabled = &value
			case "executable":
				cur.Executable = unquoteYAML(raw)
			case "routingInstructions":
				if raw == "|" || raw == ">" {
					blockKey = key
					blockIndent = indent
					block = nil
					out.Agents[agentID] = cur
					continue
				}
				cur.RoutingInstructions = unquoteYAML(raw)
			default:
				return Overlay{}, fmt.Errorf("unknown overlay key agents.%s.%s", agentID, key)
			}
			out.Agents[agentID] = cur
			continue
		}
		return Overlay{}, fmt.Errorf("line %d: unexpected indent", i+1)
	}
	flushBlock()
	return out, nil
}

func agentEmpty(agent AgentOverlay) bool {
	return agent.Enabled == nil && strings.TrimSpace(agent.Executable) == "" && strings.TrimSpace(agent.RoutingInstructions) == ""
}

func managerEmpty(manager ManagerOverlay) bool {
	return strings.TrimSpace(manager.Agent) == "" && strings.TrimSpace(manager.ThreadID) == "" && strings.TrimSpace(manager.BoundAt) == ""
}

func leadingSpaces(line string) int {
	n := 0
	for _, r := range line {
		if r != ' ' {
			break
		}
		n++
	}
	return n
}

func parseBool(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean %q", raw)
	}
}

func unquoteYAML(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if unquoted, err := strconv.Unquote(raw); err == nil {
		return unquoted
	}
	return raw
}

func yamlQuote(value string) string {
	if value == "" {
		return `""`
	}
	if needsYAMLQuote(value) {
		return strconv.Quote(value)
	}
	return value
}

func needsYAMLQuote(value string) bool {
	if strings.TrimSpace(value) != value {
		return true
	}
	return strings.ContainsAny(value, ":#\"'\n\t")
}

func trimRouting(text string) string {
	return strings.TrimRight(text, "\n")
}
