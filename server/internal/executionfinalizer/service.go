package executionfinalizer

import (
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/fleet.eggs.gd/lib/chain"
)

// NewService builds the finalization contour as a package-owned service
// constructor so composition roots depend on the finalizer boundary rather than
// assembling its internal processor chain.
func NewService(finalizer *Finalizer, in <-chan taskflow.ExecutionResult, errch chan error) chain.ChainProcessor {
	discard := make(chan taskflow.ExecutionResult)
	service := chain.NewChainProcessor(errch)
	service.AddStep(NewProcessor(finalizer, in, discard))
	return service
}
