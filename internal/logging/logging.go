package logging

import (
	"io"
	"log/slog"
	"os"
)

// Setup configures the global slog logger based on the log level and optional log file.
func Setup(level string, logFile string) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	var writer io.Writer = os.Stderr
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			writer = io.MultiWriter(os.Stderr, f)
		}
	}

	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level: logLevel,
	})
	slog.SetDefault(slog.New(handler))
}
