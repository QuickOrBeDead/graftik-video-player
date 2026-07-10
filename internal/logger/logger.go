package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	LevelTrace = slog.Level(-8)
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type FrontendSink func(level slog.Level, msg string, attrs ...slog.Attr)

type LogRotation struct {
	MaxSizeMB  int `json:"maxSizeMB"`
	MaxBackups int `json:"maxBackups"`
	MaxAgeDays int `json:"maxAgeDays"`
}

type DefaultLogger struct {
	*slog.Logger
	level        *slog.LevelVar
	frontendSink FrontendSink
	file         io.Closer
	mu           sync.Mutex
	buffered     []LogEntry
	flushed      bool
}

type LogConfig struct {
	Level       slog.Level
	LogToFile   bool
	LogDir      string
	LogFilename string
	Rotation    *LogRotation
}

func New(cfg LogConfig) *DefaultLogger {
	var handlers []slog.Handler

	l := &DefaultLogger{}
	l.level = &slog.LevelVar{}
	l.level.Set(cfg.Level)

	opts := &slog.HandlerOptions{
		Level: l.level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				if src, ok := a.Value.Any().(*slog.Source); ok {
					a.Value = slog.StringValue(fmt.Sprintf("%s:%d", filepath.Base(src.File), src.Line))
				}
			}
			return a
		},
		AddSource: true,
	}

	textHandler := slog.NewTextHandler(os.Stderr, opts)
	handlers = append(handlers, textHandler)

	if cfg.LogToFile && cfg.LogDir != "" {
		if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
			panic("logger: failed to create log dir: " + err.Error())
		}
		filename := cfg.LogFilename
		if filename == "" {
			filename = "app.log"
		}
		logPath := filepath.Join(cfg.LogDir, filename)

		var w io.WriteCloser
		maxSize, maxBackups, maxAge := 1, 10, 30
		if cfg.Rotation != nil {
			if cfg.Rotation.MaxSizeMB > 0 {
				maxSize = cfg.Rotation.MaxSizeMB
			}
			if cfg.Rotation.MaxBackups > 0 {
				maxBackups = cfg.Rotation.MaxBackups
			}
			if cfg.Rotation.MaxAgeDays > 0 {
				maxAge = cfg.Rotation.MaxAgeDays
			}
		}
		w = &lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    maxSize,
			MaxBackups: maxBackups,
			MaxAge:     maxAge,
		}

		fileHandler := slog.NewTextHandler(w, opts)
		handlers = append(handlers, fileHandler)
		l.file = w
	}

	l.Logger = slog.New(&multiHandler{handlers: handlers, logger: l})
	return l
}

func (l *DefaultLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *DefaultLogger) SetFrontendSink(sink FrontendSink) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.frontendSink = sink
	for _, entry := range l.buffered {
		if sink != nil {
			sink(entry.Level, entry.Message, entry.Attrs...)
		}
	}
	l.buffered = nil
	l.flushed = true
}

func (l *DefaultLogger) sendToFrontend(level slog.Level, msg string, attrs []slog.Attr) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.frontendSink != nil {
		l.frontendSink(level, msg, attrs...)
	} else if !l.flushed {
		entry := LogEntry{
			Time:    time.Now(),
			Level:   level,
			Message: msg,
			Attrs:   attrs,
		}
		l.buffered = append(l.buffered, entry)
		if len(l.buffered) > 1000 {
			l.buffered = l.buffered[len(l.buffered)-1000:]
		}
	}
}

type multiHandler struct {
	handlers []slog.Handler
	logger   *DefaultLogger
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if err := handler.Handle(ctx, r); err != nil {
			return err
		}
	}

	msg := r.Message
	attrs := make([]slog.Attr, 0, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a)
		return true
	})
	var source string
	if _, file, line, ok := runtime.Caller(4); ok {
		source = fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}
	if source != "" {
		attrs = append(attrs, slog.String("source", source))
	}

	h.logger.sendToFrontend(r.Level, msg, attrs)

	return nil
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers, logger: h.logger}
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithGroup(name)
	}
	return &multiHandler{handlers: handlers, logger: h.logger}
}

func (h *multiHandler) AddHandler(handler slog.Handler) {
	h.handlers = append(h.handlers, handler)
}

func ParseLevel(s string) slog.Level {
	switch s {
	case "trace":
		return LevelTrace
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelWarn
	}
}

func LevelToString(l slog.Level) string {
	switch l {
	case LevelTrace:
		return "TRACE"
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return fmt.Sprintf("LEVEL(%d)", l)
	}
}

type LogEntry struct {
	Time    time.Time
	Level   slog.Level
	Message string
	Source  string
	Attrs   []slog.Attr
}

func SyncFrontendSink(ctx context.Context, emitFn func(ctx context.Context, event string, optionalData ...interface{})) FrontendSink {
	return func(level slog.Level, msg string, attrs ...slog.Attr) {
		entry := map[string]any{
			"level":   LevelToString(level),
			"message": msg,
			"time":    time.Now().Format(time.RFC3339Nano),
		}
		if len(attrs) > 0 {
			fields := make(map[string]any)
			for _, a := range attrs {
				fields[a.Key] = a.Value.Any()
			}
			if source, ok := fields["source"]; ok {
				entry["source"] = source
				delete(fields, "source")
			}
			if len(fields) > 0 {
				entry["attrs"] = fields
			}
		}
		if emitFn != nil && ctx != nil {
			emitFn(ctx, "log", entry)
		}
	}
}
