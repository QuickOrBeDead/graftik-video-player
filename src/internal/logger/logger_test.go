package logger

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"trace", LevelTrace},
		{"debug", LevelDebug},
		{"info", LevelInfo},
		{"warn", LevelWarn},
		{"error", LevelError},
		{"unknown", LevelWarn},
		{"", LevelWarn},
	}
	for _, tc := range tests {
		got := ParseLevel(tc.input)
		if got != tc.want {
			t.Errorf("ParseLevel(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestLevelToString(t *testing.T) {
	tests := []struct {
		input slog.Level
		want  string
	}{
		{LevelTrace, "TRACE"},
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{slog.Level(999), "LEVEL(999)"},
	}
	for _, tc := range tests {
		got := LevelToString(tc.input)
		if got != tc.want {
			t.Errorf("LevelToString(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestParseLevelRoundTrip(t *testing.T) {
	known := []string{"trace", "debug", "info", "warn", "error"}
	for _, s := range known {
		level := ParseLevel(s)
		result := LevelToString(level)
		expected := strings.ToUpper(s)
		if result != expected {
			t.Errorf("round-trip %q: ParseLevel -> LevelToString = %q, want %q", s, result, expected)
		}
	}
}

func TestNew_StderrOnly(t *testing.T) {
	l := New(LogConfig{Level: LevelInfo})
	if l == nil {
		t.Fatal("New returned nil")
	}
	if l.Logger == nil {
		t.Fatal("slog.Logger is nil")
	}
	l.Close()
}

func TestNew_WithFileLogging(t *testing.T) {
	dir := t.TempDir()
	msg := "test message"
	l := New(LogConfig{
		Level:       LevelDebug,
		LogToFile:   true,
		LogDir:      dir,
		LogFilename: "test.log",
	})
	l.Info(msg)
	l.Close()

	logPath := filepath.Join(dir, "test.log")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("log file was not created")
	}

	data := parseLogFileData(t, logPath)
	if len(data) != 1 {
		t.Error("log file should have 1 line")
	}

	line := data[0]
	if v, ok := line["level"]; !ok || v != LevelToString(LevelInfo) {
		t.Error("log file line log level should be info")
	}

	if v, ok := line["msg"]; !ok || v != fmt.Sprintf("\"%s\"", msg) {
		t.Errorf("log file line log msg should be \"%s\". current is %s", msg, v)
	}
}

func TestNew_WithRotation(t *testing.T) {
	dir := t.TempDir()
	l := New(LogConfig{
		Level:       LevelInfo,
		LogToFile:   true,
		LogDir:      dir,
		LogFilename: "rot.log",
		Rotation: &LogRotation{
			MaxSizeMB:  5,
			MaxBackups: 3,
			MaxAgeDays: 7,
		},
	})
	l.Info("rotation test")
	l.Close()

	logPath := filepath.Join(dir, "rot.log")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("log file with rotation was not created")
	}

	data := parseLogFileData(t, logPath)
	if len(data) != 1 {
		t.Error("log file should have 1 line")
	}

	line := data[0]
	if v, ok := line["level"]; !ok || v != LevelToString(LevelInfo) {
		t.Error("log file line log level should be info")
	}

	if v, ok := line["msg"]; !ok || v != fmt.Sprintf("\"%s\"", "rotation test") {
		t.Errorf("log file line log msg should be \"rotation test\". current is %s", v)
	}
}

func TestNew_DefaultRotation(t *testing.T) {
	dir := t.TempDir()
	l := New(LogConfig{
		Level:       LevelInfo,
		LogToFile:   true,
		LogDir:      dir,
		LogFilename: "default_rot.log",
		Rotation:    nil,
	})
	l.Info("default rotation test")
	l.Close()

	logPath := filepath.Join(dir, "default_rot.log")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("log file with default rotation was not created")
	}

	data := parseLogFileData(t, logPath)
	if len(data) != 1 {
		t.Error("log file should have 1 line")
	}

	line := data[0]
	if v, ok := line["level"]; !ok || v != LevelToString(LevelInfo) {
		t.Error("log file line log level should be info")
	}

	if v, ok := line["msg"]; !ok || v != fmt.Sprintf("\"%s\"", "default rotation test") {
		t.Errorf("log file line log msg should be \"default rotation test\". current is %s", v)
	}
}

func TestNew_FileLogging_NoLogDir(t *testing.T) {
	l := New(LogConfig{
		Level:     LevelInfo,
		LogToFile: true,
		LogDir:    "",
	})
	l.Info("no logdir test")
	l.Close()
}

func TestNew_FileLogging_BadDir(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("New with bad log dir should panic")
		}
	}()
	New(LogConfig{
		Level:     LevelInfo,
		LogToFile: true,
		LogDir:    "/nonexistent/deeply/nested/path",
	})
}

func TestClose_WithoutFile(t *testing.T) {
	l := New(LogConfig{Level: LevelInfo})
	err := l.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestClose_WithFile(t *testing.T) {
	dir := t.TempDir()
	l := New(LogConfig{
		Level:       LevelInfo,
		LogToFile:   true,
		LogDir:      dir,
		LogFilename: "close_test.log",
	})
	err := l.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestSetFrontendSink_BuffersBefore(t *testing.T) {
	l := New(LogConfig{Level: LevelInfo})

	l.Info("before sink 1")
	l.Info("before sink 2")

	var received []string
	sink := func(level slog.Level, msg string, attrs ...slog.Attr) {
		received = append(received, msg)
	}
	l.SetFrontendSink(sink)

	if len(received) != 2 {
		t.Fatalf("expected 2 buffered entries, got %d", len(received))
	}
	if received[0] != "before sink 1" {
		t.Errorf("first entry = %q, want %q", received[0], "before sink 1")
	}
	if received[1] != "before sink 2" {
		t.Errorf("second entry = %q, want %q", received[1], "before sink 2")
	}
}

func TestSetFrontendSink_DropsBufferWhenFlushed(t *testing.T) {
	l := New(LogConfig{Level: LevelInfo})

	sink := func(level slog.Level, msg string, attrs ...slog.Attr) {}
	l.SetFrontendSink(sink)
	l.SetFrontendSink(nil)

	l.Info("after flush")

	sink2 := func(level slog.Level, msg string, attrs ...slog.Attr) {
		t.Error("should not receive entries after flushed")
	}
	l.SetFrontendSink(sink2)
}

func TestBufferOverflow(t *testing.T) {
	l := New(LogConfig{Level: LevelInfo})

	for i := 0; i < 1100; i++ {
		l.Info("msg")
	}

	var count int
	sink := func(level slog.Level, msg string, attrs ...slog.Attr) {
		count++
	}
	l.SetFrontendSink(sink)

	if count != 1000 {
		t.Errorf("expected 1000 buffered entries (capped), got %d", count)
	}
}

func TestSendToFrontend_WithSink(t *testing.T) {
	l := New(LogConfig{Level: LevelInfo})

	var receivedMsg string
	sink := func(level slog.Level, msg string, attrs ...slog.Attr) {
		receivedMsg = msg
	}
	l.SetFrontendSink(sink)
	l.Info("direct to sink")

	if receivedMsg != "direct to sink" {
		t.Errorf("received = %q, want %q", receivedMsg, "direct to sink")
	}
}

func TestSendToFrontend_NoSink(t *testing.T) {
	l := New(LogConfig{Level: LevelInfo})

	l.Info("buffered entry")

	l.mu.Lock()
	count := len(l.buffered)
	l.mu.Unlock()

	if count != 1 {
		t.Errorf("expected 1 buffered entry, got %d", count)
	}
}

func TestWriteToText(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: LevelDebug})
	l := New(LogConfig{Level: LevelDebug})
	l.textHandlers = []slog.Handler{h}
	l.mu.Lock()
	l.Logger = slog.New(&multiHandler{handlers: []slog.Handler{h}, logger: l})
	l.mu.Unlock()

	l.WriteToText(LevelInfo, "frontend msg", "key1", "val1", "key2", "val2")

	output := buf.String()
	if !strings.Contains(output, "frontend msg") {
		t.Errorf("output missing message: %s", output)
	}
	if !strings.Contains(output, "origin=frontend") {
		t.Errorf("output missing origin=frontend: %s", output)
	}
	if !strings.Contains(output, "key1=val1") {
		t.Errorf("output missing key1=val1: %s", output)
	}
}

func TestWriteToText_OddArgs(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: LevelDebug})
	l := New(LogConfig{Level: LevelDebug})
	l.textHandlers = []slog.Handler{h}
	l.mu.Lock()
	l.Logger = slog.New(&multiHandler{handlers: []slog.Handler{h}, logger: l})
	l.mu.Unlock()

	l.WriteToText(LevelInfo, "odd args", "key1", "val1", "orphan")

	output := buf.String()
	if !strings.Contains(output, "MISSING=orphan") {
		t.Errorf("output missing MISSING=orphan for odd args: %s", output)
	}
}

func TestMultiHandler_Enabled(t *testing.T) {
	level := &slog.LevelVar{}
	level.Set(LevelWarn)
	opts := &slog.HandlerOptions{Level: level}
	h1 := slog.NewTextHandler(os.Stderr, opts)
	level2 := &slog.LevelVar{}
	level2.Set(LevelDebug)
	opts2 := &slog.HandlerOptions{Level: level2}
	h2 := slog.NewTextHandler(os.Stderr, opts2)

	mh := &multiHandler{handlers: []slog.Handler{h1, h2}, logger: &DefaultLogger{}}
	if !mh.Enabled(context.Background(), LevelInfo) {
		t.Error("multiHandler should be enabled for Info when one handler allows it")
	}

	level2.Set(LevelError)
	mh2 := &multiHandler{handlers: []slog.Handler{h1, h2}, logger: &DefaultLogger{}}
	if mh2.Enabled(context.Background(), LevelInfo) {
		t.Error("multiHandler should not be enabled for Info when both handlers restrict it")
	}
}

func TestMultiHandler_Handle(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	h1 := slog.NewTextHandler(&buf1, nil)
	h2 := slog.NewTextHandler(&buf2, nil)

	l := New(LogConfig{Level: LevelDebug})
	mh := &multiHandler{handlers: []slog.Handler{h1, h2}, logger: l}

	record := slog.NewRecord(time.Now(), LevelInfo, "test handle", 0)
	if err := mh.Handle(context.Background(), record); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if !strings.Contains(buf1.String(), "test handle") {
		t.Error("handler 1 did not receive record")
	}
	if !strings.Contains(buf2.String(), "test handle") {
		t.Error("handler 2 did not receive record")
	}
}

func TestMultiHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, nil)
	l := New(LogConfig{Level: LevelDebug})
	mh := &multiHandler{handlers: []slog.Handler{h}, logger: l}

	newHandler := mh.WithAttrs([]slog.Attr{slog.String("key", "val")})
	if newHandler == nil {
		t.Fatal("WithAttrs returned nil")
	}
	if _, ok := newHandler.(*multiHandler); !ok {
		t.Fatal("WithAttrs did not return *multiHandler")
	}

	record := slog.NewRecord(time.Now(), LevelInfo, "with attrs", 0)
	if err := newHandler.Handle(context.Background(), record); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if !strings.Contains(buf.String(), "key=val") {
		t.Errorf("output missing key=val: %s", buf.String())
	}
}

func TestMultiHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, nil)
	l := New(LogConfig{Level: LevelDebug})
	mh := &multiHandler{handlers: []slog.Handler{h}, logger: l}

	newHandler := mh.WithGroup("mygroup")
	if newHandler == nil {
		t.Fatal("WithGroup returned nil")
	}
	if _, ok := newHandler.(*multiHandler); !ok {
		t.Fatal("WithGroup did not return *multiHandler")
	}
}

func TestSyncFrontendSink(t *testing.T) {
	var emitted bool
	var emittedEvent string
	var emittedData map[string]any

	emitFn := func(ctx context.Context, event string, optionalData ...any) {
		emitted = true
		emittedEvent = event
		if len(optionalData) > 0 {
			if m, ok := optionalData[0].(map[string]any); ok {
				emittedData = m
			}
		}
	}

	sink := SyncFrontendSink(context.Background(), emitFn)
	sink(LevelInfo, "hello world", slog.String("source", "test.go:42"), slog.String("extra", "data"))

	if !emitted {
		t.Fatal("emitFn was not called")
	}
	if emittedEvent != "log" {
		t.Errorf("event = %q, want %q", emittedEvent, "log")
	}
	if emittedData == nil {
		t.Fatal("emitted data is nil")
	}
	if emittedData["level"] != "INFO" {
		t.Errorf("level = %v, want INFO", emittedData["level"])
	}
	if emittedData["message"] != "hello world" {
		t.Errorf("message = %v, want %q", emittedData["message"], "hello world")
	}
	if emittedData["source"] != "test.go:42" {
		t.Errorf("source = %v, want %q", emittedData["source"], "test.go:42")
	}
	attrs, ok := emittedData["attrs"].(map[string]any)
	if !ok {
		t.Fatal("attrs is not a map")
	}
	if attrs["extra"] != "data" {
		t.Errorf("attrs[extra] = %v, want %q", attrs["extra"], "data")
	}
}

func TestSyncFrontendSink_NilEmitFn(t *testing.T) {
	sink := SyncFrontendSink(context.Background(), nil)
	sink(LevelInfo, "no emit", slog.String("k", "v"))
}

func TestSyncFrontendSink_NilContext(t *testing.T) {
	called := false
	emitFn := func(ctx context.Context, event string, optionalData ...any) {
		called = true
	}

	//lint:ignore SA1012 intentional nil context for test compatibility
	sink := SyncFrontendSink(nil, emitFn)
	sink(LevelInfo, "nil ctx")

	if called {
		t.Error("emitFn should not be called with nil context")
	}
}

func TestSyncFrontendSink_NoAttrs(t *testing.T) {
	var emittedData map[string]any
	emitFn := func(ctx context.Context, event string, optionalData ...any) {
		if len(optionalData) > 0 {
			if m, ok := optionalData[0].(map[string]any); ok {
				emittedData = m
			}
		}
	}

	sink := SyncFrontendSink(context.Background(), emitFn)
	sink(LevelError, "no attrs")

	if emittedData == nil {
		t.Fatal("emitted data is nil")
	}
	if _, hasAttrs := emittedData["attrs"]; hasAttrs {
		t.Error("attrs should not be present when no attrs are passed")
	}
}

func TestMultiHandler_AddHandler(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, nil)
	l := New(LogConfig{Level: LevelDebug})
	mh := &multiHandler{handlers: []slog.Handler{h}, logger: l}

	extraBuf := new(bytes.Buffer)
	extraHandler := slog.NewTextHandler(extraBuf, nil)
	mh.AddHandler(extraHandler)

	if len(mh.handlers) != 2 {
		t.Errorf("expected 2 handlers after AddHandler, got %d", len(mh.handlers))
	}

	record := slog.NewRecord(time.Now(), LevelInfo, "added handler", 0)
	if err := mh.Handle(context.Background(), record); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if !strings.Contains(extraBuf.String(), "added handler") {
		t.Error("extra handler did not receive record")
	}
}

func TestWriteToTextHandlers(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: LevelDebug})
	l := New(LogConfig{Level: LevelDebug})
	l.textHandlers = []slog.Handler{h}

	l.WriteToTextHandlers(LevelWarn, "warn msg", slog.String("foo", "bar"))

	output := buf.String()
	if !strings.Contains(output, "warn msg") {
		t.Errorf("output missing warn msg: %s", output)
	}
	if !strings.Contains(output, "foo=bar") {
		t.Errorf("output missing foo=bar: %s", output)
	}
}

func parseLogFileData(t *testing.T, logPath string) []map[string]string {
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Error("log file cannot be read")
	}

	scanner := bufio.NewScanner(strings.NewReader(string(content)))

	data := []map[string]string{}
	for scanner.Scan() {
		line := scanner.Text()
		m := map[string]string{}
		inQuotes := false
		fields := strings.FieldsFuncSeq(line, func(r rune) bool {
			if r == '"' {
				inQuotes = !inQuotes
				return false
			}

			return r == ' ' && !inQuotes
		})

		for f := range fields {
			k, v, found := strings.Cut(f, "=")
			if !found {
				panic(fmt.Sprintf("could not split log line field. no = in %s", f))
			}
			m[k] = v
		}

		data = append(data, m)
	}
	return data
}
