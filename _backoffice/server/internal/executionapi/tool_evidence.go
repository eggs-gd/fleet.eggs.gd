package executionapi

import (
	"sort"
	"strings"
	"time"
)

const (
	maxStoredToolCalls   = 48
	maxToolSummaryChars  = 160
	toolEvidenceSourceRPC = "provider_rpc"
	toolEvidenceSourceLog = "session_log"
)

// ToolCallEvidence is one compact observation that a named tool ran.
// It is derived from provider streams/logs, not from worker self-report text.
type ToolCallEvidence struct {
	Name     string `json:"name"`
	CalledAt string `json:"called_at,omitempty"`
	Source   string `json:"source,omitempty"`
	Status   string `json:"status,omitempty"`
	Summary  string `json:"summary,omitempty"`
}

// ToolUsageEvidence is the per-session tool audit projected to the dashboard.
type ToolUsageEvidence struct {
	Profiles  []string           `json:"profiles,omitempty"`
	Required  []string           `json:"required,omitempty"`
	Used      []ToolCallEvidence `json:"used,omitempty"`
	Missing   []string           `json:"missing,omitempty"`
	Warning   string             `json:"warning,omitempty"`
	UpdatedAt string             `json:"updated_at,omitempty"`
}

// HasMissing reports whether any required tools were never observed.
func (e ToolUsageEvidence) HasMissing() bool {
	return len(e.Missing) > 0
}

// InitToolUsage builds the initial required/missing set for a new session.
func InitToolUsage(input RequirementInput, at time.Time) ToolUsageEvidence {
	profiles := SelectRequirementProfiles(input)
	required := RequiredToolsForProfiles(profiles)
	return ProjectToolUsage(profiles, required, nil, at)
}

// ProjectToolUsage recomputes missing tools and warning text from used evidence.
func ProjectToolUsage(profiles, required []string, used []ToolCallEvidence, at time.Time) ToolUsageEvidence {
	used = compactToolCalls(used)
	missing := missingRequiredTools(required, used)
	evidence := ToolUsageEvidence{
		Profiles:  uniqueSorted(profiles),
		Required:  uniqueSorted(required),
		Used:      used,
		Missing:   missing,
		UpdatedAt: formatToolEvidenceTime(at),
	}
	evidence.Warning = ToolUsageWarning(evidence)
	return evidence
}

// MergeToolCalls folds newly observed calls into existing evidence.
func MergeToolCalls(existing ToolUsageEvidence, calls []ToolCallEvidence, at time.Time) ToolUsageEvidence {
	if len(calls) == 0 && existing.UpdatedAt != "" {
		return existing
	}
	merged := append([]ToolCallEvidence{}, existing.Used...)
	merged = append(merged, calls...)
	return ProjectToolUsage(existing.Profiles, existing.Required, merged, at)
}

// RefreshToolUsageMissing recomputes missing/warning without changing used calls.
func RefreshToolUsageMissing(existing ToolUsageEvidence, at time.Time) ToolUsageEvidence {
	return ProjectToolUsage(existing.Profiles, existing.Required, existing.Used, at)
}

// ToolUsageWarning returns operator-facing text when required tools are missing.
func ToolUsageWarning(evidence ToolUsageEvidence) string {
	if len(evidence.Required) == 0 || len(evidence.Missing) == 0 {
		return ""
	}
	return "Missing required tool evidence: " + strings.Join(evidence.Missing, ", ") +
		". Agent may have ignored AGENTS.md MCP/tool policy."
}

// WithToolUsageWarning appends a compact tool-evidence warning to a summary.
func WithToolUsageWarning(summary string, evidence ToolUsageEvidence) string {
	warning := strings.TrimSpace(evidence.Warning)
	if warning == "" {
		warning = ToolUsageWarning(evidence)
	}
	if warning == "" {
		return summary
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return warning
	}
	if strings.Contains(summary, warning) {
		return summary
	}
	return summary + "\n\n" + warning
}

func missingRequiredTools(required []string, used []ToolCallEvidence) []string {
	seen := map[string]bool{}
	for _, call := range used {
		name := NormalizeToolName(call.Name)
		if name == "" {
			continue
		}
		seen[name] = true
		for _, alias := range ToolNameAliases(name) {
			seen[alias] = true
		}
	}
	missing := make([]string, 0, len(required))
	for _, name := range uniqueSorted(required) {
		if seen[name] {
			continue
		}
		matched := false
		for _, alias := range ToolNameAliases(name) {
			if seen[alias] {
				matched = true
				break
			}
		}
		if !matched {
			missing = append(missing, name)
		}
	}
	return missing
}

func compactToolCalls(calls []ToolCallEvidence) []ToolCallEvidence {
	if len(calls) == 0 {
		return nil
	}
	type key struct {
		name   string
		source string
		status string
	}
	index := map[key]int{}
	out := make([]ToolCallEvidence, 0, len(calls))
	for _, call := range calls {
		call.Name = NormalizeToolName(call.Name)
		if call.Name == "" {
			continue
		}
		call.Summary = truncateToolSummary(call.Summary)
		call.Source = strings.TrimSpace(call.Source)
		call.Status = strings.TrimSpace(call.Status)
		call.CalledAt = strings.TrimSpace(call.CalledAt)
		k := key{name: call.Name, source: call.Source, status: call.Status}
		if i, ok := index[k]; ok {
			prev := out[i]
			if call.CalledAt != "" {
				prev.CalledAt = call.CalledAt
			}
			if call.Summary != "" {
				prev.Summary = call.Summary
			}
			out[i] = prev
			continue
		}
		index[k] = len(out)
		out = append(out, call)
	}
	if len(out) > maxStoredToolCalls {
		out = out[len(out)-maxStoredToolCalls:]
	}
	return out
}

func truncateToolSummary(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) <= maxToolSummaryChars {
		return text
	}
	return text[:maxToolSummaryChars]
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func formatToolEvidenceTime(at time.Time) string {
	if at.IsZero() {
		at = time.Now()
	}
	return at.UTC().Format(time.RFC3339)
}
