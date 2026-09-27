package executionapi

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

type TaskContext struct {
	ID           string
	Ref          string
	Title        string
	Status       string
	Type         string
	Priority     int
	ProjectID    string
	WorkspaceID  string
	Repository   string
	RelativePath string
	WorkingDir   string
	Assignee     string
	LaunchMode   string
	Body         string
}

type Plan struct {
	Agent           string
	Backend         string
	Command         []string
	WorkingDir      string
	Prompt          string
	InitialInput    string
	VisibilityClass string
	LiveReady       bool
	LiveNotes       string
}

// Agent is one provider adapter. Every adapter states what it can do, so
// launch, recovery and Settings read the same answer.
type Agent interface {
	Name() string
	Plan(task TaskContext) (Plan, error)
	// Capabilities is what a person and Fleet can do with a session of this
	// provider: see it, continue it, detect its end.
	Capabilities() ProviderCapabilities
	// SessionReuse says whether Fleet may resume an earlier session on the
	// next task, and how well that has been confirmed.
	SessionReuse() SessionReuseCapability
}

type Registry struct {
	agents map[string]Agent
}

func NewRegistry() *Registry {
	return &Registry{agents: map[string]Agent{}}
}

func (registry *Registry) Register(agent Agent) {
	registry.agents[strings.ToLower(agent.Name())] = agent
}

// Agent returns the registered adapter with this name.
func (registry *Registry) Agent(name string) (Agent, bool) {
	agent, ok := registry.agents[strings.ToLower(name)]
	return agent, ok
}

// Agents returns every registered adapter, in name order.
func (registry *Registry) Agents() []Agent {
	names := make([]string, 0, len(registry.agents))
	for name := range registry.agents {
		names = append(names, name)
	}
	sort.Strings(names)
	agents := make([]Agent, 0, len(names))
	for _, name := range names {
		agents = append(agents, registry.agents[name])
	}
	return agents
}

func (registry *Registry) Plan(name string, task TaskContext) (Plan, error) {
	agent, ok := registry.agents[strings.ToLower(name)]
	if !ok {
		return Plan{}, fmt.Errorf("%w: %s", ErrUnknownAgent, name)
	}
	return agent.Plan(task)
}

var ErrUnknownAgent = errors.New("unknown agent")
