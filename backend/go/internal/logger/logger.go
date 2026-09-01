// Package logger zap 日志装配(对齐 Luna-CY 模板 internal/logger)
package logger

import (
	stdlog "log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// L 全局日志实例(Init 后可用);zap.L() 经 ReplaceGlobals 同步可用
var L *zap.Logger
var fileSink *redactingSink

// Init 按运行模式初始化 zap: release=JSON 生产格式,其余=彩色开发格式
func Init(mode string) error {
	var l *zap.Logger
	var err error
	if mode == "release" {
		l, err = zap.NewProduction()
	} else {
		l, err = zap.NewDevelopment()
	}
	if err != nil {
		return err
	}
	L = l
	zap.ReplaceGlobals(l)
	return nil
}

// Options controls the desktop file logger. DataDir is the application data
// directory, not an arbitrary client supplied path.
type Options struct {
	Mode    string
	DataDir string
	BootID  string
	Console bool
}

type redactingSink struct {
	mu sync.Mutex
	f  *lumberjack.Logger
}

func (s *redactingSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.f.Write([]byte(RedactText(string(p)))); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (s *redactingSink) Sync() error { return nil }
func (s *redactingSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.f.Close()
}

// InitWithOptions installs a JSON logger with bounded local rotation. Desktop
// launches do not have a terminal, so the file sink is the authoritative log.
func InitWithOptions(opts Options) error {
	logDir := filepath.Join(opts.DataDir, "logs")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return err
	}
	file := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, "app.log"),
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     14,
		Compress:   true,
	}
	sink := &redactingSink{f: file}
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.TimeKey = "timestamp"
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	cores := []zapcore.Core{zapcore.NewCore(zapcore.NewJSONEncoder(encCfg), sink, zap.InfoLevel)}
	if opts.Console {
		cores = append(cores, zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), zapcore.AddSync(os.Stderr), zap.InfoLevel))
	}
	l := zap.New(zapcore.NewTee(cores...), zap.Fields(zap.String("boot_id", opts.BootID)))
	L = l
	fileSink = sink
	stdlog.SetOutput(sink)
	zap.ReplaceGlobals(l)
	return nil
}

// LogDir returns the conventional subdirectory name used by desktop callers.
func LogDir(dataDir string) string { return filepath.Join(dataDir, "logs") }

var (
	secretHeader = regexp.MustCompile(`(?i)(authorization\s*:\s*bearer\s+|x-alchemy-token\s*:\s*|api[_-]?key\s*[=:]\s*)([^\s,;]+)`)
	secretURL    = regexp.MustCompile(`(?i)([?&](?:token|api[_-]?key|authorization|signature)=)[^&\s]+`)
	dataURI      = regexp.MustCompile(`(?i)data:image/[^;]+;base64,[^\s]+`)
)

// RedactText is used before writing untrusted process output to a support log.
// It intentionally leaves stable error codes and ordinary diagnostic text.
func RedactText(input string) string {
	out := secretHeader.ReplaceAllString(input, `$1[REDACTED]`)
	out = secretURL.ReplaceAllString(out, `$1[REDACTED]`)
	out = dataURI.ReplaceAllString(out, "data:image/[REDACTED]")
	// Avoid accidentally retaining common key-shaped values in free-form errors.
	for _, prefix := range []string{"sk-", "sk_"} {
		for {
			idx := strings.Index(strings.ToLower(out), prefix)
			if idx < 0 {
				break
			}
			end := idx
			for end < len(out) && !strings.ContainsRune(" \t\r\n,;\"'", rune(out[end])) {
				end++
			}
			out = out[:idx] + "[REDACTED]" + out[end:]
		}
	}
	return out
}

// Sync 刷新日志缓冲区,进程退出前调用
func Sync() {
	if L != nil {
		_ = L.Sync()
	}
	if fileSink != nil {
		_ = fileSink.Close()
		fileSink = nil
	}
}
