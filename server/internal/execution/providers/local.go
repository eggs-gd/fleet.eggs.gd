package providers

import (
	"strings"
	"sync"
)

// LocalAgent is the machine overlay for one registered executor. Launch and
// discovery read this; Settings persists it in core.local.yaml.
type LocalAgent struct {
	Enabled    *bool
	Executable string
}

var (
	localMu     sync.RWMutex
	localAgents = map[string]LocalAgent{}
)

func ApplyLocalAgents(agents map[string]LocalAgent) {
	localMu.Lock()
	defer localMu.Unlock()
	localAgents = map[string]LocalAgent{}
	for id, agent := range agents {
		localAgents[strings.ToLower(strings.TrimSpace(id))] = agent
	}
}

func AgentEnabled(name string) bool {
	localMu.RLock()
	defer localMu.RUnlock()
	agent, ok := localAgents[strings.ToLower(strings.TrimSpace(name))]
	if !ok || agent.Enabled == nil {
		return true
	}
	return *agent.Enabled
}

func localAgent(name string) LocalAgent {
	localMu.RLock()
	defer localMu.RUnlock()
	return localAgents[strings.ToLower(strings.TrimSpace(name))]
}

func localExecutable(name string) string {
	return strings.TrimSpace(localAgent(name).Executable)
}
