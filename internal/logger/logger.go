package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

const (
	LevelTrace = slog.Level(-8)
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

type FrontendSink func(level slog.Level, msg string, attrs ...slog.Attr)

type Logger struct {
	*slog.Logger
	level        slog.Leveler
	frontendSink FrontendSink
	file         *os.File
	mu           sync.Mutex
	buffered     []LogEntry
	flushed      bool
}

type LogConfig struct {
	Level       slog.Level
	LogToFile   bool
	LogDir      string
	LogFilename string
}

func New(cfg LogConfig) *Logger {
	var handlers []slog.Handler

	opts := &slog.HandlerOptions{
		Level: cfg.Level,
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

	l := &Logger{
		level: cfg.Level,
	}

	if cfg.LogToFile && cfg.LogDir != "" {
		if err := os.MkdirAll(cfg.LogDir, 0755); err == nil {
			filename := cfg.LogFilename
			if filename == "" {
				filename = "app.log"
			}
			logPath := filepath.Join(cfg.LogDir, filename)
			file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err == nil {
				fileHandler := slog.NewTextHandler(file, opts)
				handlers = append(handlers, fileHandler)
				l.file = file
			}
		}
	}

	l.Logger = slog.New(&multiHandler{handlers: handlers, logger: l})
	return l
}

func (l *Logger) SetLevel(level slog.Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *Logger) SetFrontendSink(sink FrontendSink) {
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

func (l *Logger) AddFileHandler(path string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	handler := slog.NewTextHandler(file, &slog.HandlerOptions{
		Level:     l.level,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				if src, ok := a.Value.Any().(*slog.Source); ok {
					a.Value = slog.StringValue(fmt.Sprintf("%s:%d", filepath.Base(src.File), src.Line))
				}
			}
			return a
		},
	})
	if mh, ok := l.Handler().(*multiHandler); ok {
		mh.handlers = append(mh.handlers, handler)
	}
	l.file = file
	return nil
}

func (l *Logger) sendToFrontend(level slog.Level, msg string, attrs []slog.Attr) {
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
	logger   *Logger
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
		return LevelInfo
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
