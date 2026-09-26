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
	entries, err := os.ReadDir(workDir)
	if err != nil {
		if os.IsNotExist(err) {
			return report, nil
		}
		return report, err
	}
	type card struct {
		id, path, text string
		hasFrontmatter bool
	}
	var cards []card
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
			return report, err
		}
		text := string(raw)
		_, err = mdfile.ListItems(text, "tag")
		cards = append(cards, card{id: entry.Name(), path: path, text: text, hasFrontmatter: err == nil})
	}
	sort.Slice(cards, func(i, j int) bool { return cards[i].id < cards[j].id })

	taken := map[string]bool{projecttag.Inbox: true}
	users := map[string][]string{}
	for _, c := range cards {
		tag := projecttag.FromCard(c.text)
		if tag == "" {
			continue
		}
		if !projecttag.Valid(tag) {
			report.Problems = append(report.Problems, fmt.Sprintf("project %s has tag %q, which is not 2-6 capital letters or digits starting with a letter", c.id, tag))
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
			report.Problems = append(report.Problems, fmt.Sprintf("tag %s is used by %s; give one of them another tag", tag, strings.Join(users[tag], ", ")))
		}
	}

	for _, c := range cards {
		if projecttag.FromCard(c.text) != "" {
			continue
		}
		if !c.hasFrontmatter {
			report.Problems = append(report.Problems, fmt.Sprintf("project %s has no frontmatter, so it cannot get a tag; add a `---` block with `tag:` to %s", c.id, c.path))
			continue
		}
		tag := projecttag.Generate(c.id, taken)
		if tag == "" {
			return report, fmt.Errorf("no free ref tag left for project %s", c.id)
		}
		next, err := mdfile.SetScalar(c.text, "tag", jsonString(tag))
		if err != nil {
			return report, fmt.Errorf("card %s: %w", c.path, err)
		}
		if err := writeFileAtomic(c.path, []byte(next)); err != nil {
			return report, err
		}
		taken[tag] = true
		report.Tagged = append(report.Tagged, c.id)
	}
	return report, nil
}
