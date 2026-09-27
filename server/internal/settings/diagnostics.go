package settings

import (
	"fmt"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/execution/providers"
)

func inspectDiagnostics(projects Projects, agents Agents, manager Manager, integrations Integrations, sessions SessionCounts) Diagnostics {
	_ = manager
	health := []HealthItem{
		registryHealth(projects),
		storageHealth(projects),
		agentHealth(agents),
		{ID: "manager", Label: "Manager", Status: "ok", Detail: "HTTP command API up; STT/LLM unconfigured; no provider session"},
		mcpHealth(integrations),
		sessionHealth(sessions),
		{ID: "scheduler", Label: "Scheduler", Status: "ok", Detail: "1-1-1 locks; launch skipped only when dry-run"},
	}
	return Diagnostics{
		Health:   health,
		Sessions: sessions,
		Issues:   collectIssues(projects, agents, integrations, sessions),
		Notes: []string{
			"Session counts use RuntimeSession.IsActive plus orphaned_tasks — the same hub as Work → Sessions.",
			"Active here excludes HITL; Work's live strip currently includes waiting_input sessions in runtime_sessions.",
			"Repair actions are not exposed. Recheck probes discovery only; Rescan is Settings → Projects.",
		},
	}
}

func registryHealth(projects Projects) HealthItem {
	for _, root := range projects.Roots {
		if root.Kind == "registry_scan" && !root.Reachable {
			return HealthItem{ID: "registry", Label: "Project registry", Status: "warn", Detail: "scan root unreachable: " + root.Path}
		}
	}
	if len(projects.Roots) == 0 {
		return HealthItem{ID: "registry", Label: "Project registry", Status: "warn", Detail: "no roots"}
	}
	return HealthItem{ID: "registry", Label: "Project registry", Status: "ok"}
}

func storageHealth(projects Projects) HealthItem {
	for _, path := range projects.DataPaths {
		if path.Name == "Tasks" && !path.Exists {
			return HealthItem{ID: "storage", Label: "Task storage", Status: "error", Detail: path.Path + " missing"}
		}
		if path.Name == "Tasks" && !path.Writable {
			return HealthItem{ID: "storage", Label: "Task storage", Status: "warn", Detail: path.Path + " not writable"}
		}
	}
	return HealthItem{ID: "storage", Label: "Task storage", Status: "ok", Detail: projects.TaskBackend.Active}
}

func agentHealth(agents Agents) HealthItem {
	warns := []string{}
	for _, agent := range agents.Providers {
		if !agent.Enabled.Value {
			continue
		}
		if agent.Status != providers.AgentReady {
			warns = append(warns, agent.Name+" "+agent.Status)
		}
	}
	if len(warns) > 0 {
		return HealthItem{ID: "agents", Label: "Agent registry", Status: "warn", Detail: strings.Join(warns, "; ")}
	}
	return HealthItem{ID: "agents", Label: "Agent registry", Status: "ok"}
}

func mcpHealth(integrations Integrations) HealthItem {
	missing := 0
	for _, server := range integrations.MCP {
		if server.Configured && !server.Detected {
			missing++
		}
	}
	if missing > 0 {
		return HealthItem{ID: "mcp", Label: "MCP integrations", Status: "warn", Detail: fmt.Sprintf("%d configured server(s) not detected", missing)}
	}
	if len(integrations.MCP) == 0 {
		return HealthItem{ID: "mcp", Label: "MCP integrations", Status: "ok", Detail: "no .mcp.json servers"}
	}
	return HealthItem{ID: "mcp", Label: "MCP integrations", Status: "ok"}
}

func sessionHealth(sessions SessionCounts) HealthItem {
	if sessions.Orphaned > 0 {
		return HealthItem{
			ID:     "sessions",
			Label:  "Sessions",
			Status: "warn",
			Detail: fmt.Sprintf("%d orphaned (%d resumable)", sessions.Orphaned, sessions.OrphanedResumable),
		}
	}
	return HealthItem{ID: "sessions", Label: "Sessions", Status: "ok"}
}

func collectIssues(projects Projects, agents Agents, integrations Integrations, sessions SessionCounts) []Issue {
	var issues []Issue
	for _, agent := range agents.Providers {
		if !agent.Enabled.Value {
			continue
		}
		if agent.Status == providers.AgentNonCanonical {
			issues = append(issues, Issue{Severity: "warn", Code: "agent_non_canonical", Message: agent.Name + " standalone CLI missing or non-canonical: " + firstNonEmpty(agent.EffectiveExecutable.Value, agent.Error), Section: "agents"})
		}
		if agent.Status == providers.AgentNotFound {
			issues = append(issues, Issue{Severity: "warn", Code: "agent_not_found", Message: agent.Name + " not found. Expected: " + agent.Expected, Section: "agents"})
		}
	}
	for _, root := range projects.Roots {
		if !root.Reachable {
			issues = append(issues, Issue{Severity: "warn", Code: "root_unavailable", Message: "Project root unavailable: " + root.Path, Section: "projects"})
		}
	}
	for _, path := range projects.DataPaths {
		if path.Name == "Data root" && path.Exists && !path.Writable {
			issues = append(issues, Issue{Severity: "warn", Code: "data_not_writable", Message: "Data directory not writable: " + path.Path, Section: "projects"})
		}
	}
	for _, server := range integrations.MCP {
		if server.Configured && !server.Detected {
			issues = append(issues, Issue{Severity: "warn", Code: "mcp_undetected", Message: "MCP " + server.Name + " not detected. Expected: " + server.Expected, Section: "integrations"})
		}
	}
	if sessions.Orphaned > 0 {
		issues = append(issues, Issue{Severity: "warn", Code: "orphaned_sessions", Message: fmt.Sprintf("%d orphaned sessions (%d resumable)", sessions.Orphaned, sessions.OrphanedResumable), Section: "diagnostics"})
	}
	return issues
}
