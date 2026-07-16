package hls

import (
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

type fakeLogger struct{}

func (f *fakeLogger) Debug(msg string, args ...any)             {}
func (f *fakeLogger) Info(msg string, args ...any)              {}
func (f *fakeLogger) Warn(msg string, args ...any)              {}
func (f *fakeLogger) Error(msg string, args ...any)             {}
func (f *fakeLogger) WriteToText(level slog.Level, msg string, args ...any) {}

func findFFmpeg(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := filepath.Join("..", "..", "build", runtime.GOOS, "bin")
	candidates := []string{"ffmpeg", "ffmpeg.exe"}
	for _, name := range candidates {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Auto-download if not found
	script := filepath.Join("..", "..", "build", runtime.GOOS, "download-ffmpeg" + scriptExt())
	cmd := execCommand(script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("download-ffmpeg failed: %v\n%s", err, out)
	}

	for _, name := range candidates {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	t.Fatalf("ffmpeg not found after running download script at %s", dir)
	return ""
}

func newTestEngine(t *testing.T, ffmpegPath string) *Engine {
	t.Helper()
	return NewEngine(ffmpegPath, t.TempDir(), &fakeLogger{})
}
