package platform

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// InitLogging creates logDir, opens access.log and app.log, and configures slog to write
// JSON lines to stderr and app.log. access.log is returned for the HTTP access middleware.
func InitLogging(logDir string, debug bool) (accessLog *os.File, appLog *os.File, err error) {
	if err = os.MkdirAll(logDir, 0750); err != nil {
		return nil, nil, err
	}

	accessPath := filepath.Join(logDir, "access.log")
	appPath := filepath.Join(logDir, "app.log")

	accessLog, err = os.OpenFile(accessPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		return nil, nil, err
	}
	appLog, err = os.OpenFile(appPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		_ = accessLog.Close()
		return nil, nil, err
	}

	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(io.MultiWriter(os.Stderr, appLog), &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))

	slog.Info("logging initialized", "log_dir", logDir, "debug", debug)
	return accessLog, appLog, nil
}
