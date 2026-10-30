package projectscan

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/mdfile"
	"github.com/eggs-gd/fleet.eggs.gd/internal/projecttag"
)

// assignTags gives every project card that has no `tag` one, so task refs can
// be TAG-N. A tag a person wrote is never changed. Duplicate and malformed
// tags, and cards it cannot tag, are reported in report.Problems for a person
// to fix. Cards are handled in id order so the result does not depend on the
// order the filesystem lists them.
func assignTags(workDir string, report CardReport) (CardReport, error) {
	cards, err := readTagCards(workDir)
	if err != nil {
		return report, err
	}
	if cards == nil {
		return report, nil
	}
	taken := map[string]bool{projecttag.Inbox: true}
	report.Problems = append(report.Problems, existingTagProblems(cards, taken)...)

	for _, c := range cards {
		if projecttag.FromCard(c.text) != "" {
			continue
		}
		if !c.hasFrontmatter {
			report.Problems = append(report.Problems, fmt.Sprintf("project %s has no frontmatter, so it cannot get a tag; add a `---` block with `tag:` to %s", c.id, c.path))
			continue
		}
		tag, err := writeNewTag(c, taken)
		if err != nil {
			return report, err
		}
		taken[tag] = true
		report.Tagged = append(report.Tagged, c.id)
	}
	return report, nil
}

type tagCard struct {
	id, path, text string
	hasFrontmatter bool
}

// readTagCards reads every project card under workDir in id order. It returns
// nil cards when the work directory does not exist.
func readTagCards(workDir string) ([]tagCard, error) {
	entries, err := os.ReadDir(workDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	cards := []tagCard{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(workDir, entry.Name(), "PROJECT.md")
		raw, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		text := string(raw)
		_, err = mdfile.ListItems(text, "tag")
		cards = append(cards, tagCard{id: entry.Name(), path: path, text: text, hasFrontmatter: err == nil})
	}
	sort.Slice(cards, func(i, j int) bool { return cards[i].id < cards[j].id })
	return cards, nil
}

// existingTagProblems marks the tags cards already carry as taken and reports
// the malformed and the shared ones.
func existingTagProblems(cards []tagCard, taken map[string]bool) []string {
	var problems []string
	users := map[string][]string{}
	for _, c := range cards {
		tag := projecttag.FromCard(c.text)
		if tag == "" {
			continue
		}
		if !projecttag.Valid(tag) {
			problems = append(problems, fmt.Sprintf("project %s has tag %q, which is not 2-6 capital letters or digits starting with a letter", c.id, tag))
			continue
		}
		taken[tag] = true
		users[tag] = append(users[tag], c.id)
	}
	tags := make([]string, 0, len(users))
	for tag := range users {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	for _, tag := range tags {
		if len(users[tag]) > 1 {
			problems = append(problems, fmt.Sprintf("tag %s is used by %s; give one of them another tag", tag, strings.Join(users[tag], ", ")))
		}
	}
	return problems
}

// writeNewTag generates a free tag for the card and writes it into the card.
func writeNewTag(c tagCard, taken map[string]bool) (string, error) {
	tag := projecttag.Generate(c.id, taken)
	if tag == "" {
		return "", fmt.Errorf("no free ref tag left for project %s", c.id)
	}
	next, err := mdfile.SetScalar(c.text, "tag", jsonString(tag))
	if err != nil {
		return "", fmt.Errorf("card %s: %w", c.path, err)
	}
	if err := writeFileAtomic(c.path, []byte(next)); err != nil {
		return "", err
	}
	return tag, nil
}
