// Package logging provides structured logging utilities for stats-agent-team.
// It uses log/slog with context propagation following the pattern from CLAUDE.md.
package logging

import (
	"context"
	"log/slog"
	"os"

	"github.com/grokify/mogo/log/slogutil"
)

// DefaultLogger returns the default logger configured for the application.
// Uses JSON handler for production-ready structured logging.
func DefaultLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// JSONLogger returns a logger with JSON output format.
func JSONLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// DebugLogger returns a logger with debug level enabled.
func DebugLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

// WithLogger returns a new context with the given logger attached.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return slogutil.ContextWithLogger(ctx, logger)
}

// FromContext retrieves the logger from context.
// If no logger is found, returns the default logger.
func FromContext(ctx context.Context) *slog.Logger {
	return slogutil.LoggerFromContext(ctx, DefaultLogger())
}

// WithComponent returns a logger with a component attribute for agent identification.
func WithComponent(logger *slog.Logger, component string) *slog.Logger {
	return logger.With("component", component)
}

// NewAgentLogger creates a logger configured for a specific agent.
func NewAgentLogger(agentName string) *slog.Logger {
	return WithComponent(DefaultLogger(), agentName)
}

// NewAgentContext creates a context with an agent-specific logger.
func NewAgentContext(agentName string) context.Context {
	return WithLogger(context.Background(), NewAgentLogger(agentName))
}
