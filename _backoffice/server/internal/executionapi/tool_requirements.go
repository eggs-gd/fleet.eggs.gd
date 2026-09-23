package executionapi

import (
	"path/filepath"
	"strings"
)

// Known requirement profiles for AGENTS.md tool policy.
const (
	ProfileGo           = "go"
	ProfileSvelte       = "svelte"
	ProfileArchitecture = "architecture"
	ProfileMCPDocs      = "mcp_docs"
)

// Canonical required tool names (MCP / agent tool surface).
const (
	ToolGoDiagnostics     = "go_diagnostics"
	ToolGoVulncheck       = "go_vulncheck"
	ToolSvelteAutofixer   = "svelte-autofixer"
	ToolListSections      = "list-sections"
	ToolGetDocumentation  = "get-documentation"
	ToolFindPatterns      = "find_patterns"
	ToolFindSimilarCode   = "find_similar_code"
)

// RequirementInput is the task/repo context used to select required tools.
type RequirementInput struct {
	TaskType     string
	Title        string
	Body         string
	Repository   string
	WorkingDir   string
	Repositories []string
}

// SelectRequirementProfiles chooses which tool-policy profiles apply.
//
// Selection is deterministic and based on task type plus path/content signals.
// See `_docs/AGENT_TOOL_EVIDENCE.md` for the operator-facing explanation.
func SelectRequirementProfiles(input RequirementInput) []string {
	text := requirementText(input)
	taskType := strings.ToLower(strings.TrimSpace(input.TaskType))

	goSignals := containsAny(text,
		".go", "_backoffice/server", "/server/", "go test", "gopls", "go.mod", "go sum",
	)
	svelteSignals := containsAny(text,
		".svelte", "_backoffice/view", "/view/", "sveltekit", "svelte-autofixer", "styles.css",
	)
	archSignals := taskType == "research" || containsAny(text,
		"refactor", "architecture", "module boundary", "module boundaries",
		"design pattern", "split corechain", "ownership boundary",
	)
	mcpSignals := svelteSignals || containsAny(text,
		"mcp", "list-sections", "get-documentation", "documentation tool", "svelte mcp",
	)

	// Only default to Go inside Core / backoffice-shaped trees. Non-Go target
	// repos (e.g. eggs-gd-prod Python/Svelte apps) must not inherit go_diagnostics.
	coreTree := containsAny(text,
		"core.eggs.gd", "core-eggs-gd", "_backoffice/server", "_backoffice/view",
	)

	switch taskType {
	case "feature", "bug", "maintenance":
		if !goSignals && !svelteSignals && coreTree {
			// Coding tasks in the Core monorepo default to Go when signals are weak.
			goSignals = true
		}
	case "review":
		if !goSignals && !svelteSignals && !archSignals {
			archSignals = true
		}
	}

	profiles := make([]string, 0, 4)
	if goSignals {
		profiles = append(profiles, ProfileGo)
	}
	if svelteSignals {
		profiles = append(profiles, ProfileSvelte)
	}
	if archSignals {
		profiles = append(profiles, ProfileArchitecture)
	}
	if mcpSignals {
		profiles = append(profiles, ProfileMCPDocs)
	}
	return uniqueSorted(profiles)
}

// RequiredToolsForProfiles returns the union of required tool names for profiles.
func RequiredToolsForProfiles(profiles []string) []string {
	required := make([]string, 0, 8)
	for _, profile := range profiles {
		switch profile {
		case ProfileGo:
			required = append(required, ToolGoDiagnostics)
		case ProfileSvelte:
			required = append(required, ToolSvelteAutofixer)
		case ProfileArchitecture:
			required = append(required, ToolFindPatterns)
		case ProfileMCPDocs:
			required = append(required, ToolListSections)
		}
	}
	return uniqueSorted(required)
}

// RequirementInputFromTask builds RequirementInput from common task fields.
func RequirementInputFromTask(taskType, title, body, repository, workingDir string, repositories []string) RequirementInput {
	return RequirementInput{
		TaskType:     taskType,
		Title:        title,
		Body:         body,
		Repository:   repository,
		WorkingDir:   workingDir,
		Repositories: append([]string{}, repositories...),
	}
}

// NormalizeToolName canonicalizes observed tool identifiers.
func NormalizeToolName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.Trim(name, "`\"'")
	if name == "" {
		return ""
	}
	lower := strings.ToLower(name)
	switch lower {
	case "go_diagnostics", "godiagnostics", "go-diagnostics":
		return ToolGoDiagnostics
	case "go_vulncheck", "govulncheck", "go-vulncheck":
		return ToolGoVulncheck
	case "svelte-autofixer", "svelte_autofixer", "svelteautofixer":
		return ToolSvelteAutofixer
	case "list-sections", "list_sections", "listsections":
		return ToolListSections
	case "get-documentation", "get_documentation", "getdocumentation":
		return ToolGetDocumentation
	case "find_patterns", "find-patterns", "findpatterns":
		return ToolFindPatterns
	case "find_similar_code", "find-similar-code", "findsimilarcode":
		return ToolFindSimilarCode
	case "get_pattern_details", "get-pattern-details":
		return "get_pattern_details"
	case "search_patterns", "search-patterns":
		return "search_patterns"
	case "go_references", "go-references":
		return "go_references"
	case "go_symbol_references", "go-symbol-references":
		return "go_symbol_references"
	case "go_package_api", "go-package-api":
		return "go_package_api"
	default:
		return lower
	}
}

// ToolNameAliases returns alternate names that satisfy a required tool.
func ToolNameAliases(name string) []string {
	switch NormalizeToolName(name) {
	case ToolFindPatterns:
		return []string{"search_patterns", "get_pattern_details"}
	case ToolListSections:
		return []string{ToolGetDocumentation}
	default:
		return nil
	}
}

func requirementText(input RequirementInput) string {
	parts := []string{
		input.TaskType,
		input.Title,
		input.Body,
		input.Repository,
		input.WorkingDir,
		filepath.Base(input.WorkingDir),
	}
	parts = append(parts, input.Repositories...)
	return strings.ToLower(strings.Join(parts, "\n"))
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if needle != "" && strings.Contains(text, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}
