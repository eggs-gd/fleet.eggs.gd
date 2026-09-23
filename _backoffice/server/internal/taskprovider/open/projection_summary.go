package open

import (
	"strings"

	"github.com/eggs-gd/core.eggs.gd/internal/taskprovider"
)

func taskProjectionChangeSummary(before taskprovider.Task, after taskprovider.Task, hadBefore bool) []string {
	if !hadBefore {
		return []string{"projected=new_task"}
	}

	var changes []string
	if before.Status != after.Status {
		changes = append(changes, "status="+before.Status+"->"+after.Status)
	}
	if before.Assignee != after.Assignee {
		changes = append(changes, "assignee="+before.Assignee+"->"+after.Assignee)
	}
	if before.Priority != after.Priority {
		changes = append(changes, "priority="+intString(before.Priority)+"->"+intString(after.Priority))
	}
	if before.UpdatedAt != after.UpdatedAt {
		changes = append(changes, "updated_at="+before.UpdatedAt+"->"+after.UpdatedAt)
	}
	if before.Launch.Agent != after.Launch.Agent {
		changes = append(changes, "launch.agent="+before.Launch.Agent+"->"+after.Launch.Agent)
	}
	if len(before.Comments) != len(after.Comments) {
		changes = append(changes, "comments="+intString(len(before.Comments))+"->"+intString(len(after.Comments)))
	}

	beforeDone, beforeTotal := acceptanceProgress(before.Body)
	afterDone, afterTotal := acceptanceProgress(after.Body)
	if beforeDone != afterDone || beforeTotal != afterTotal {
		changes = append(changes, "acceptance="+intString(beforeDone)+"/"+intString(beforeTotal)+"->"+intString(afterDone)+"/"+intString(afterTotal))
	}

	changes = append(changes, sectionChanges(before.Body, after.Body)...)
	if len(changes) == 0 && before.Body != after.Body {
		changes = append(changes, "body_changed=true")
	}
	return changes
}

func acceptanceProgress(body string) (int, int) {
	lower := strings.ToLower(body)
	done := strings.Count(lower, "- [x]")
	total := done + strings.Count(lower, "- [ ]")
	return done, total
}

func sectionChanges(before string, after string) []string {
	beforeSections := markdownSections(before)
	afterSections := markdownSections(after)
	var changes []string
	for section := range afterSections {
		if !beforeSections[section] {
			changes = append(changes, "section_added="+section)
		}
	}
	for section := range beforeSections {
		if !afterSections[section] {
			changes = append(changes, "section_removed="+section)
		}
	}
	return changes
}

func markdownSections(body string) map[string]bool {
	sections := map[string]bool{}
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ") {
			continue
		}
		section := strings.TrimSpace(strings.TrimPrefix(line, "## "))
		if section != "" {
			sections[section] = true
		}
	}
	return sections
}
