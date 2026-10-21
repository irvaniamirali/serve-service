package logging

import (
	"log/slog"
	"os"
)

// New creates the process-wide structured JSON logger.
func New() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}
