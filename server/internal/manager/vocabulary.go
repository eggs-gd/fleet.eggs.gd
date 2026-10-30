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
	// Milestones are the workspace card's milestones (flat, with a state).
	// They live on the workspace, so a repository project of a group uses its
	// parent's (WorkspaceID).
	Milestones  []board.Milestone `json:"milestones,omitempty"`
	WorkspaceID string            `json:"workspace_id,omitempty"`
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
	v := &vocabularyBuilder{seenTerm: map[string]bool{}, terms: make([]string, 0, 128), techSeen: map[string]string{}}
	for _, term := range fixedOperationalTerms {
		v.addTerm(term)
	}

	workspaces := v.workspaceEntities(view)
	projects := v.projectEntities(view)
	for _, ws := range view.Workspaces {
		v.collectTech(ws.Technology.EffectiveTags...)
		v.collectTech(ws.Technology.DetectedTags...)
	}
	for _, project := range view.Projects {
		v.collectTech(project.Technology.EffectiveTags...)
		v.collectTech(project.Technology.DetectedTags...)
	}
	repositories := v.repositoryEntities(view)

	technologies := make([]string, 0, len(v.techSeen))
	for _, display := range v.techSeen {
		technologies = append(technologies, display)
	}
	sort.Strings(technologies)

	return Vocabulary{
		Terms:         v.terms,
		Prompt:        buildVocabularyPrompt(projects, workspaces, technologies),
		Projects:      projects,
		Workspaces:    workspaces,
		Repositories:  repositories,
		Technologies:  technologies,
		GeneratedFrom: "board projection/workspaces/technologies + operational terms",
	}
}

// vocabularyBuilder collects the de-duplicated terms and technology names.
type vocabularyBuilder struct {
	seenTerm map[string]bool
	terms    []string
	techSeen map[string]string
}

func (v *vocabularyBuilder) addTerm(term string) {
	term = strings.TrimSpace(term)
	if term == "" {
		return
	}
	key := strings.ToLower(term)
	if v.seenTerm[key] {
		return
	}
	v.seenTerm[key] = true
	v.terms = append(v.terms, term)
}

func (v *vocabularyBuilder) collectTech(tags ...string) {
	for _, tag := range tags {
		display := technologyDisplay(tag)
		if display == "" {
			continue
		}
		key := strings.ToLower(display)
		if _, ok := v.techSeen[key]; ok {
			continue
		}
		v.techSeen[key] = display
		v.addTerm(display)
	}
}

func (v *vocabularyBuilder) workspaceEntities(view BoardView) []VocabularyEntity {
	out := make([]VocabularyEntity, 0, len(view.Workspaces))
	for _, ws := range view.Workspaces {
		entity := VocabularyEntity{
			ID:            strings.TrimSpace(ws.ID),
			Title:         firstNonEmpty(ws.Title, ws.ID),
			Summary:       truncateRunes(strings.TrimSpace(ws.Summary), summaryEntityMaxRunes),
			Aliases:       uniqueFold(collectAliases(append([]string{ws.ID, ws.Title}, ws.Repositories...)...)),
			Kind:          "workspace",
			SummarySource: firstNonEmpty(ws.SummarySource, "generated"),
			Milestones:    ws.Milestones,
		}
		if entity.ID == "" && entity.Title == "" {
			continue
		}
		out = append(out, entity)
		addEntityTerms(v.addTerm, entity)
		addTechnologyTerms(v.addTerm, ws.Technology)
	}
	sortByTitle(out)
	return out
}

func (v *vocabularyBuilder) projectEntities(view BoardView) []VocabularyEntity {
	out := make([]VocabularyEntity, 0, len(view.Projects))
	for _, project := range view.Projects {
		entity := VocabularyEntity{
			ID:            strings.TrimSpace(project.ID),
			Title:         firstNonEmpty(project.Title, project.ID),
			Summary:       truncateRunes(strings.TrimSpace(project.Summary), summaryEntityMaxRunes),
			Aliases:       uniqueFold(collectAliases(append([]string{project.ID, project.Title}, project.Repositories...)...)),
			Kind:          "project",
			SummarySource: firstNonEmpty(project.SummarySource, "generated"),
			WorkspaceID:   project.WorkspaceID,
		}
		if entity.ID == "" && entity.Title == "" {
			continue
		}
		out = append(out, entity)
		addEntityTerms(v.addTerm, entity)
		addTechnologyTerms(v.addTerm, project.Technology)
	}
	sortByTitle(out)
	return out
}

func (v *vocabularyBuilder) repositoryEntities(view BoardView) []VocabularyEntity {
	out := make([]VocabularyEntity, 0, len(view.Registry.Repositories))
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
		out = append(out, entity)
		addEntityTerms(v.addTerm, entity)
		v.collectTech(repo.EffectiveTags...)
		v.collectTech(repo.DetectedTags...)
		v.collectTech(technologyProfileTags(repo.Effective)...)
		v.collectTech(technologyProfileTags(repo.Detected)...)
	}
	sortByTitle(out)
	return out
}

func sortByTitle(entities []VocabularyEntity) {
	sort.Slice(entities, func(i, j int) bool {
		return strings.ToLower(entities[i].Title) < strings.ToLower(entities[j].Title)
	})
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
	b.WriteString("Fleet Manager vocabulary, generated dynamically from registry and workspace state.\n")
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
	if len(entity.Milestones) > 0 {
		b.WriteString(" Milestones: ")
		for i, milestone := range entity.Milestones {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(b, "%s (%s)", firstNonEmpty(milestone.Title, milestone.ID), milestone.State)
		}
		b.WriteByte('.')
	}
	b.WriteByte('\n')
}

func truncateRunes(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || value == "" {
		return value
	}
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	trimmed := strings.TrimSpace(string(runes[:limit]))
	if cut := strings.LastIndexAny(trimmed, " .,;"); cut > limit/2 {
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
