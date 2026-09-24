package logger_test

import (
	"testing"

	l "github.com/eggs-gd/fleet.eggs.gd/lib/logger"
	"github.com/eggs-gd/fleet.eggs.gd/lib/logger/decorators"
)

func TestPerceptrailLoggerAPI(t *testing.T) {
	logger := l.NewLogger(l.InfoLevel, &decorators.GontrollerDecorator{})

	logger.Info("logger smoke test", l.String("stage", "test"), l.Int("count", 1))
	logger.Named("Core").Warn("named logger smoke test", l.String("task", "CORE-38"))
	logger.DisableService("Core")
	logger.EnableService("Core")
}
