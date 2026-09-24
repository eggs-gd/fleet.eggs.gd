package providers

import "github.com/eggs-gd/fleet.eggs.gd/internal/executionapi"

func NewRegistry() *executionapi.Registry {
	registry := executionapi.NewRegistry()
	registry.Register(Claude{})
	registry.Register(Codex{})
	registry.Register(Cursor{})
	registry.Register(Gemini{})
	return registry
}
