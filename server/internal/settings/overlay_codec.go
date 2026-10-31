package settings

import (
	"fmt"
	"strconv"
	"strings"
)

func EncodeOverlay(overlay Overlay) []byte {
	var b strings.Builder
	b.WriteString("# Machine-local Fleet overlay. Do not commit.\n")
	writeQuoted(&b, "", "sessionTimeout", strings.TrimSpace(overlay.SessionTimeout))
	writeQuoted(&b, "", "launch", strings.TrimSpace(overlay.Launch))
	if roots := normalizeScanRoots(overlay.ScanRoots); len(roots) > 0 {
		b.WriteString("scanRoots:\n")
		for _, root := range roots {
			fmt.Fprintf(&b, "  - %s\n", yamlQuote(root))
		}
	}
	encodeAgents(&b, overlay.Agents)
	if !managerEmpty(overlay.Manager) {
		b.WriteString("manager:\n")
		writeQuoted(&b, "  ", "agent", overlay.Manager.Agent)
		writeQuoted(&b, "  ", "threadId", overlay.Manager.ThreadID)
		writeQuoted(&b, "  ", "workspace", overlay.Manager.Workspace)
		writeQuoted(&b, "  ", "boundAt", overlay.Manager.BoundAt)
	}
	return []byte(b.String())
}

// writeQuoted writes `key: value`, or nothing when the value is blank.
func writeQuoted(b *strings.Builder, indent string, key string, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	fmt.Fprintf(b, "%s%s: %s\n", indent, key, yamlQuote(value))
}

func encodeAgents(b *strings.Builder, agents map[string]AgentOverlay) {
	wroteAgents := false
	for _, id := range knownAgentIDs {
		agent, ok := agents[id]
		if !ok || agentEmpty(agent) {
			continue
		}
		if !wroteAgents {
			b.WriteString("agents:\n")
			wroteAgents = true
		}
		fmt.Fprintf(b, "  %s:\n", id)
		if agent.Enabled != nil {
			fmt.Fprintf(b, "    enabled: %v\n", *agent.Enabled)
		}
		writeQuoted(b, "    ", "executable", agent.Executable)
		if strings.TrimSpace(agent.RoutingInstructions) != "" {
			b.WriteString("    routingInstructions: |\n")
			for _, line := range strings.Split(agent.RoutingInstructions, "\n") {
				fmt.Fprintf(b, "      %s\n", line)
			}
		}
	}
}

func ParseOverlay(data []byte) (Overlay, error) {
	p := &overlayParser{out: Overlay{Agents: map[string]AgentOverlay{}}}
	for i, line := range strings.Split(string(data), "\n") {
		p.lineNo = i + 1
		if err := p.parseLine(strings.TrimRight(line, "\r")); err != nil {
			return Overlay{}, err
		}
	}
	p.flushBlock()
	return p.out, nil
}

// overlayParser reads the small YAML subset EncodeOverlay writes, line by line.
type overlayParser struct {
	out     Overlay
	lineNo  int
	section string
	agentID string

	// A routingInstructions block scalar being collected.
	blockKey    string
	blockIndent int
	block       []string
}

func (p *overlayParser) flushBlock() {
	if p.blockKey != "" && p.agentID != "" {
		cur := p.out.Agents[p.agentID]
		cur.RoutingInstructions = trimRouting(strings.Join(p.block, "\n"))
		p.out.Agents[p.agentID] = cur
	}
	p.blockKey = ""
	p.block = nil
}

// continueBlock takes line into the open block scalar; false means the block
// ended and the line is an ordinary one.
func (p *overlayParser) continueBlock(line string) bool {
	if strings.TrimSpace(line) == "" {
		p.block = append(p.block, "")
		return true
	}
	if leadingSpaces(line) <= p.blockIndent {
		p.flushBlock()
		return false
	}
	prefix := p.blockIndent + 2
	if len(line) >= prefix {
		p.block = append(p.block, line[prefix:])
	} else {
		p.block = append(p.block, strings.TrimLeft(line, " "))
	}
	return true
}

func (p *overlayParser) parseLine(line string) error {
	if p.blockKey != "" && p.continueBlock(line) {
		return nil
	}
	trim := strings.TrimSpace(line)
	if trim == "" || strings.HasPrefix(trim, "#") {
		return nil
	}
	indent := leadingSpaces(line)
	if indent == 2 && p.section == "scanRoots" && strings.HasPrefix(trim, "-") {
		item := strings.TrimSpace(strings.TrimPrefix(trim, "-"))
		if item == "" {
			return fmt.Errorf("line %d: empty scan root", p.lineNo)
		}
		p.out.ScanRoots = append(p.out.ScanRoots, unquoteYAML(item))
		return nil
	}
	key, raw, ok := strings.Cut(trim, ":")
	if !ok {
		return fmt.Errorf("line %d: expected key: value", p.lineNo)
	}
	key = strings.TrimSpace(key)
	raw = strings.TrimSpace(raw)

	switch {
	case indent == 0:
		return p.topLevel(key, raw)
	case indent == 2 && p.section == "manager":
		return p.managerField(key, raw)
	case indent == 2 && p.section == "agents":
		return p.agentHeader(key)
	case indent == 4 && p.agentID != "":
		return p.agentField(key, raw, indent)
	}
	return fmt.Errorf("line %d: unexpected indent", p.lineNo)
}

func (p *overlayParser) topLevel(key string, raw string) error {
	p.agentID = ""
	switch key {
	case "sessionTimeout":
		p.section = ""
		p.out.SessionTimeout = unquoteYAML(raw)
	case "launch":
		p.section = ""
		p.out.Launch = unquoteYAML(raw)
	case "scanRoots":
		p.section = "scanRoots"
		if raw != "" {
			p.out.ScanRoots = append(p.out.ScanRoots, unquoteYAML(raw))
		}
	case "agents":
		p.section = "agents"
	case "manager":
		p.section = "manager"
	default:
		return fmt.Errorf("unknown overlay key %q", key)
	}
	return nil
}

func (p *overlayParser) managerField(key string, raw string) error {
	switch key {
	case "agent":
		p.out.Manager.Agent = unquoteYAML(raw)
	case "threadId":
		p.out.Manager.ThreadID = unquoteYAML(raw)
	case "workspace":
		p.out.Manager.Workspace = unquoteYAML(raw)
	case "boundAt":
		p.out.Manager.BoundAt = unquoteYAML(raw)
	default:
		return fmt.Errorf("unknown overlay key manager.%s", key)
	}
	return nil
}

func (p *overlayParser) agentHeader(key string) error {
	if !isKnownAgent(key) {
		return fmt.Errorf("unknown agent %q", key)
	}
	p.agentID = key
	if _, ok := p.out.Agents[key]; !ok {
		p.out.Agents[key] = AgentOverlay{}
	}
	return nil
}

func (p *overlayParser) agentField(key string, raw string, indent int) error {
	cur := p.out.Agents[p.agentID]
	switch key {
	case "enabled":
		value, err := parseBool(raw)
		if err != nil {
			return fmt.Errorf("agents.%s.enabled: %w", p.agentID, err)
		}
		cur.Enabled = &value
	case "executable":
		cur.Executable = unquoteYAML(raw)
	case "routingInstructions":
		if raw == "|" || raw == ">" {
			p.blockKey = key
			p.blockIndent = indent
			p.block = nil
			p.out.Agents[p.agentID] = cur
			return nil
		}
		cur.RoutingInstructions = unquoteYAML(raw)
	default:
		return fmt.Errorf("unknown overlay key agents.%s.%s", p.agentID, key)
	}
	p.out.Agents[p.agentID] = cur
	return nil
}

func agentEmpty(agent AgentOverlay) bool {
	return agent.Enabled == nil && strings.TrimSpace(agent.Executable) == "" && strings.TrimSpace(agent.RoutingInstructions) == ""
}

func managerEmpty(manager ManagerOverlay) bool {
	return strings.TrimSpace(manager.Agent) == "" && strings.TrimSpace(manager.ThreadID) == "" && strings.TrimSpace(manager.Workspace) == "" && strings.TrimSpace(manager.BoundAt) == ""
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
