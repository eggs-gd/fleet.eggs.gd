package manager

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/eggs-gd/fleet.eggs.gd/internal/board"
)

const (
	summaryPromptMaxRunes = 160
	summaryEntityMaxRunes = 240
)

// Fixed operational terms that are not derived from project/registry state.
// Project names and stack labels come from Core state dynamically.
var fixedOperationalTerms = []string{
	"Core",
	"Codex",
	"Claude",
	"Cursor",
	"Gemini",
	"MCP",
	"Plane",
	"backlog",
	"todo",
	"needs review",
	"needs rework",
	"blocked",
	"done",
	"archived",
}

// Known STT-friendly spellings for technology tags.
var technologyDisplayNames = map[string]string{
	"csharp":         "C#",
	"dotnet":         ".NET",
	"docker-compose": "Docker Compose",
	"go":             "Go",
	"golang":         "Go",
	"javascript":     "JavaScript",
	"typescript":     "TypeScript",
	"nodejs":         "Node.js",
	"node":           "Node.js",
	"python":         "Python",
	"svelte":         "Svelte",
	"react":          "React",
	"vue":            "Vue",
	"rust":           "Rust",
	"docker":         "Docker",
	"mcp":            "MCP",
}

// VocabularyEntity is one workspace/project/repository entry for STT + manager
// context. Summary and aliases help the model catch synonyms, not just exact
// titles.
type VocabularyEntity struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Summary string   `json:"summary,omitempty"`
	Aliases []string `json:"aliases,omitempty"`
	Kind    string   `json:"kind"`
	// SummarySource is "generated" (the scanner's own guess, may be wrong or
	// boilerplate) or "confirmed" (a person or the Manager approved it via
	// manager_describe). Empty for kinds that carry no card, such as agents.
	SummarySource string `json:"summary_source,omitempty"`
}

// Vocabulary is the dynamic STT / manager-context payload.
type Vocabulary struct {
	Terms         []string           `json:"terms"`
	Prompt        string             `json:"prompt"`
	Projects      []VocabularyEntity `json:"projects"`
	Workspaces    []VocabularyEntity `json:"workspaces"`
	Repositories  []VocabularyEntity `json:"repositories"`
	Technologies  []string           `json:"technologies"`
	GeneratedFrom string             `json:"generated_from"`
}

// BuildVocabulary derives transcription and classification hints from runtime
// Core state: projects/workspaces with short explanations, repository aliases,
// and technologies present in context.
func BuildVocabulary(view BoardView) Vocabulary {
	seenTerm := map[string]bool{}
	terms := make([]string, 0, 128)
	addTerm := func(term string) {
		term = strings.TrimSpace(term)
		if term == "" {
			return
		}
		key := strings.ToLower(term)
		if seenTerm[key] {
			return
		}
		seenTerm[key] = true
		terms = append(terms, term)
	}

	for _, term := range fixedOperationalTerms {
		addTerm(term)
	}

	workspaces := make([]VocabularyEntity, 0, len(view.Workspaces))
	for _, ws := range view.Workspaces {
		entity := VocabularyEntity{
			ID:            strings.TrimSpace(ws.ID),
			Title:         firstNonEmpty(ws.Title, ws.ID),
			Summary:       truncateRunes(strings.TrimSpace(ws.Summary), summaryEntityMaxRunes),
			Aliases:       uniqueFold(collectAliases(append([]string{ws.ID, ws.Title}, ws.Repositories...)...)),
			Kind:          "workspace",
			SummarySource: firstNonEmpty(ws.SummarySource, "generated"),
		}
		if entity.ID == "" && entity.Title == "" {
			continue
		}
		workspaces = append(workspaces, entity)
		addEntityTerms(addTerm, entity)
		addTechnologyTerms(addTerm, ws.Technology)
	}
	sort.Slice(workspaces, func(i, j int) bool {
		return strings.ToLower(workspaces[i].Title) < strings.ToLower(workspaces[j].Title)
	})

	projects := make([]VocabularyEntity, 0, len(view.Projects))
	for _, project := range view.Projects {
		entity := VocabularyEntity{
			ID:            strings.TrimSpace(project.ID),
			Title:         firstNonEmpty(project.Title, project.ID),
			Summary:       truncateRunes(strings.TrimSpace(project.Summary), summaryEntityMaxRunes),
			Aliases:       uniqueFold(collectAliases(append([]string{project.ID, project.Title}, project.Repositories...)...)),
			Kind:          "project",
			SummarySource: firstNonEmpty(project.SummarySource, "generated"),
		}
		if entity.ID == "" && entity.Title == "" {
			continue
		}
		projects = append(projects, entity)
		addEntityTerms(addTerm, entity)
		addTechnologyTerms(addTerm, project.Technology)
	}
	sort.Slice(projects, func(i, j int) bool {
		return strings.ToLower(projects[i].Title) < strings.ToLower(projects[j].Title)
	})

	repositories := make([]VocabularyEntity, 0, len(view.Registry.Repositories))
	techSeen := map[string]string{}
	collectTech := func(tags ...string) {
		for _, tag := range tags {
			display := technologyDisplay(tag)
			if display == "" {
				continue
			}
			key := strings.ToLower(display)
			if _, ok := techSeen[key]; ok {
				continue
			}
			techSeen[key] = display
			addTerm(display)
		}
	}

	for _, ws := range view.Workspaces {
		collectTech(ws.Technology.EffectiveTags...)
		collectTech(ws.Technology.DetectedTags...)
	}
	for _, project := range view.Projects {
		collectTech(project.Technology.EffectiveTags...)
		collectTech(project.Technology.DetectedTags...)
	}

	for _, repo := range view.Registry.Repositories {
		entity := VocabularyEntity{
			ID:      firstNonEmpty(repo.ID, repo.RelativePath, repo.Name),
			Title:   firstNonEmpty(repo.Name, repoBaseName(repo.RelativePath), repo.RelativePath),
			Summary: "",
			Aliases: uniqueFold(collectAliases(repo.ID, repo.Name, repo.RelativePath, repoBaseName(repo.RelativePath))),
			Kind:    "repository",
		}
		if entity.ID == "" && entity.Title == "" {
			continue
		}
		repositories = append(repositories, entity)
		addEntityTerms(addTerm, entity)
		collectTech(repo.EffectiveTags...)
		collectTech(repo.DetectedTags...)
		collectTech(technologyProfileTags(repo.Effective)...)
		collectTech(technologyProfileTags(repo.Detected)...)
	}
	sort.Slice(repositories, func(i, j int) bool {
		return strings.ToLower(repositories[i].Title) < strings.ToLower(repositories[j].Title)
	})

	technologies := make([]string, 0, len(techSeen))
	for _, display := range techSeen {
		technologies = append(technologies, display)
	}
	sort.Strings(technologies)

	return Vocabulary{
		Terms:         terms,
		Prompt:        buildVocabularyPrompt(projects, workspaces, technologies),
		Projects:      projects,
		Workspaces:    workspaces,
		Repositories:  repositories,
		Technologies:  technologies,
		GeneratedFrom: "board projection/workspaces/technologies + operational terms",
	}
}

func addEntityTerms(add func(string), entity VocabularyEntity) {
	add(entity.Title)
	add(entity.ID)
	for _, alias := range entity.Aliases {
		add(alias)
		add(repoBaseName(alias))
	}
}

func addTechnologyTerms(add func(string), summary board.TechnologySummary) {
	for _, tag := range summary.EffectiveTags {
		add(technologyDisplay(tag))
	}
	for _, tag := range summary.DetectedTags {
		add(technologyDisplay(tag))
	}
}

func collectAliases(values ...string) []string {
	out := make([]string, 0, len(values)*2)
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
		if base := repoBaseName(value); base != "" && !strings.EqualFold(base, value) {
			out = append(out, base)
		}
	}
	return out
}

func technologyProfileTags(profile board.TechnologyProfile) []string {
	out := make([]string, 0, 16)
	out = append(out, profile.Languages...)
	out = append(out, profile.Frameworks...)
	out = append(out, profile.Runtimes...)
	out = append(out, profile.Tooling...)
	return out
}

func technologyDisplay(tag string) string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return ""
	}
	key := strings.ToLower(tag)
	if display, ok := technologyDisplayNames[key]; ok {
		return display
	}
	parts := strings.FieldsFunc(tag, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func buildVocabularyPrompt(projects, workspaces []VocabularyEntity, technologies []string) string {
	var b strings.Builder
	b.WriteString("Core Manager vocabulary, generated dynamically from registry and workspace state.\n")
	b.WriteString("Prefer the listed spellings. Use each short description to recognize synonyms and informal names.\n\n")

	if len(projects) > 0 {
		b.WriteString("Projects:\n")
		for _, project := range projects {
			writeEntityLine(&b, project)
		}
		b.WriteByte('\n')
	}
	if len(workspaces) > 0 {
		b.WriteString("Workspaces:\n")
		for _, workspace := range workspaces {
			writeEntityLine(&b, workspace)
		}
		b.WriteByte('\n')
	}
	if len(technologies) > 0 {
		b.WriteString("Technologies in context: ")
		b.WriteString(strings.Join(technologies, ", "))
		b.WriteString(".\n\n")
	}
	b.WriteString("Operational terms: ")
	b.WriteString(strings.Join(fixedOperationalTerms, ", "))
	b.WriteString(".")
	return b.String()
}

func writeEntityLine(b *strings.Builder, entity VocabularyEntity) {
	title := firstNonEmpty(entity.Title, entity.ID)
	b.WriteString("- ")
	b.WriteString(title)
	if entity.ID != "" && !strings.EqualFold(entity.ID, title) {
		fmt.Fprintf(b, " (id: %s)", entity.ID)
	}
	summary := truncateRunes(entity.Summary, summaryPromptMaxRunes)
	if summary != "" {
		b.WriteString(": ")
		b.WriteString(summary)
	}
	aliases := make([]string, 0, len(entity.Aliases))
	for _, alias := range entity.Aliases {
		if strings.EqualFold(alias, title) || strings.EqualFold(alias, entity.ID) {
			continue
		}
		aliases = append(aliases, alias)
	}
	if len(aliases) > 0 {
		b.WriteString(" Aliases: ")
		b.WriteString(strings.Join(aliases, ", "))
		b.WriteByte('.')
	}
	b.WriteByte('\n')
}

func truncateRunes(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || value == "" {
		return value
	}
	if utf8.RuneCountInString(value) <= max {
		return value
	}
	runes := []rune(value)
	trimmed := strings.TrimSpace(string(runes[:max]))
	if cut := strings.LastIndexAny(trimmed, " .,;"); cut > max/2 {
		trimmed = strings.TrimRight(trimmed[:cut], " .,;")
	}
	return trimmed + "…"
}

func repoBaseName(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = strings.ReplaceAll(path, "\\", "/")
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
