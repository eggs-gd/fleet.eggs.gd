package executionapi

import (
	"errors"
	"fmt"
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

type Agent interface {
	Name() string
	Plan(task TaskContext) (Plan, error)
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

func (registry *Registry) Plan(name string, task TaskContext) (Plan, error) {
	agent, ok := registry.agents[strings.ToLower(name)]
	if !ok {
		return Plan{}, fmt.Errorf("%w: %s", ErrUnknownAgent, name)
	}
	return agent.Plan(task)
}

var ErrUnknownAgent = errors.New("unknown agent")
