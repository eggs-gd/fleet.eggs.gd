package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/eggs-gd/fleet.eggs.gd/internal/settings"
)

// Codes a Manager session request can fail with. The dashboard shows the message
// and keys its help off the code.
const (
	sessionInvalid       = "invalid"
	sessionNotInstalled  = "not_installed"
	sessionAuthRequired  = "auth_required"
	sessionTimedOut      = "timeout"
	sessionProviderError = "failed"
)

type setupResponse struct {
	Needed bool `json:"needed"`
	// Providers says which agents are installed here, so the dashboard offers
	// only what can work.
	Providers []settings.ProviderAvailability `json:"providers,omitempty"`
}

var (
	urlPattern  = regexp.MustCompile(`https?://\S+`)
	authPattern = regexp.MustCompile(`(?i)authenticat|log ?in\b|sign ?in\b|not logged|unauthorized|api key|credentials`)
)

// classifyManagerError turns what a provider printed into a short message for the
// person. The provider's own output can be long and can carry sign-in links, so
// it goes to the log and never to the dashboard.
func classifyManagerError(agent string, err error) (code, message string) {
	name := providerName(agent)
	text := err.Error()
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return sessionTimedOut, fmt.Sprintf("%s did not answer in %d seconds. Open %s, make sure you are signed in, then try again.", name, int(managerSessionTimeout.Seconds()), name)
	case errors.Is(err, exec.ErrNotFound) || strings.Contains(text, "executable file not found") || strings.Contains(text, "no such file or directory"):
		return sessionNotInstalled, fmt.Sprintf("%s was not found on this computer. Install it, then try again.", name)
	case authPattern.MatchString(text):
		return sessionAuthRequired, fmt.Sprintf("%s is not signed in. Open %s, sign in, then try again.", name, name)
	}
	return sessionProviderError, fmt.Sprintf("%s could not start the session: %s (details are in the Fleet log)", name, firstLine(text))
}

var providerNames = map[string]string{"claude": "Claude", "codex": "Codex", "cursor": "Cursor", "gemini": "Gemini"}

func providerName(agent string) string {
	if name, ok := providerNames[agent]; ok {
		return name
	}
	if agent == "" {
		return "The provider"
	}
	return agent
}

func firstLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(urlPattern.ReplaceAllString(line, "[link removed]"))
		if line == "" {
			continue
		}
		if utf8.RuneCountInString(line) > 200 {
			line = string([]rune(line)[:200]) + "…"
		}
		return line
	}
	return "no details"
}

// writeManagerSessionError answers a failed Manager session request with a
// JSON error the dashboard can show as is.
func writeManagerSessionError(w http.ResponseWriter, agent string, err error) {
	code, message := sessionInvalid, err.Error()
	var invalid errString
	if !errors.As(err, &invalid) {
		code, message = classifyManagerError(agent, err)
		fmt.Fprintf(os.Stderr, "manager session for %q failed: %v\n", agent, err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message, "code": code})
}
