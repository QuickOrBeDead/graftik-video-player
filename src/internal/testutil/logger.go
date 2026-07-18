package testutil

import (
	"log/slog"

	graftikLogger "github.com/QuickOrBeDead/graftik-video-player/internal/logger"
)

type NopLogger struct{}

func (NopLogger) Debug(msg string, args ...any)                         {}
func (NopLogger) Info(msg string, args ...any)                          {}
func (NopLogger) Warn(msg string, args ...any)                          {}
func (NopLogger) Error(msg string, args ...any)                         {}
func (NopLogger) WriteToText(level slog.Level, msg string, args ...any) {}

var _ graftikLogger.Logger = NopLogger{}
