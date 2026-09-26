package tasklifecycle

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// AgentAssignees are the AI workers Fleet can launch. Every other assignee is a
// person, and a person is a worker only when the operator registered them as
// Fleet/<name>.md.
var AgentAssignees = []string{"claude", "codex", "cursor", "gemini"}

// Unassigned is the assignee of a task nobody has taken.
const Unassigned = "unassigned"

var assigneeSlug = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// Fleet files that describe the fleet, not a worker.
var fleetDocs = map[string]bool{"readme": true, "routing": true, "launch_policy": true}

// IsAgent reports whether name is one of the launchable AI workers.
func IsAgent(name string) bool {
	for _, agent := range AgentAssignees {
		if agent == name {
			return true
		}
	}
	return false
}

// RosterPerson reports whether Fleet/<name>.md exists: a person registered as a
// worker. Names are case-folded, and the fleet's own documents do not count.
func RosterPerson(root, name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if strings.TrimSpace(root) == "" || !assigneeSlug.MatchString(name) || fleetDocs[name] || IsAgent(name) {
		return false
	}
	entries, err := os.ReadDir(filepath.Join(root, "Fleet"))
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(entry.Name(), name+".md") {
			return true
		}
	}
	return false
}

// KnownAssignee reports whether name may be given a task: unassigned, an agent,
// or a person in the Fleet roster.
func KnownAssignee(root, name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	return name == Unassigned || IsAgent(name) || RosterPerson(root, name)
}

// ValidAssigneeName reports whether name has the shape of an assignee. It does
// not check the roster.
func ValidAssigneeName(name string) bool {
	return assigneeSlug.MatchString(strings.ToLower(strings.TrimSpace(name)))
}
