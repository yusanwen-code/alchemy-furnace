package engineproc

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/alchemy-furnace/server/internal/logger"
	"gopkg.in/natefinch/lumberjack.v2"
)

// pythonLogWriter keeps the embedded interpreter's two output streams on disk
// even when the desktop app was launched without a terminal.
type pythonLogWriter struct {
	mu sync.Mutex
	f  *lumberjack.Logger
}

func newPythonLogWriter(path string) (*pythonLogWriter, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	return &pythonLogWriter{f: &lumberjack.Logger{
		Filename: path, MaxSize: 10, MaxBackups: 5, MaxAge: 14, Compress: true,
	}}, nil
}

func (w *pythonLogWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	redacted := logger.RedactText(string(p))
	_, err := w.f.Write([]byte(redacted))
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *pythonLogWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.f.Close()
}
