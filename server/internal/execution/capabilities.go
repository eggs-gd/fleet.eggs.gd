package execution

import "github.com/eggs-gd/fleet.eggs.gd/internal/executionapi"

const (
	BackendBackgroundRemote = executionapi.BackendBackgroundRemote
	BackendCodexAppServer   = executionapi.BackendCodexAppServer
	BackendCursorVisible    = executionapi.BackendCursorVisible
	BackendGeminiHeadless   = executionapi.BackendGeminiHeadless
)

type (
	SessionReuseLevel      = executionapi.SessionReuseLevel
	CapabilityFlag         = executionapi.CapabilityFlag
	SessionReuseCapability = executionapi.SessionReuseCapability
	ProviderCapabilities   = executionapi.ProviderCapabilities
)

const (
	SessionReuseVerified    = executionapi.SessionReuseVerified
	SessionReuseUnverified  = executionapi.SessionReuseUnverified
	SessionReuseUnsupported = executionapi.SessionReuseUnsupported

	CapabilityYes     = executionapi.CapabilityYes
	CapabilityNo      = executionapi.CapabilityNo
	CapabilityUnknown = executionapi.CapabilityUnknown
)

func ProviderCapabilityFlags(provider string, backend string) ProviderCapabilities {
	return executionapi.ProviderCapabilityFlags(provider, backend)
}

func ProviderSessionReuseCapability(backend string) SessionReuseCapability {
	return executionapi.ProviderSessionReuseCapability(backend)
}
