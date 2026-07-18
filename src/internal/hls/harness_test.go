package hls

import (
	"log/slog"
	"testing"

	"github.com/QuickOrBeDead/graftik-video-player/internal/testutil"
)

type fakeLogger struct{}

func (f *fakeLogger) Debug(msg string, args ...any)             {}
func (f *fakeLogger) Info(msg string, args ...any)              {}
func (f *fakeLogger) Warn(msg string, args ...any)              {}
func (f *fakeLogger) Error(msg string, args ...any)             {}
func (f *fakeLogger) WriteToText(level slog.Level, msg string, args ...any) {}

func findFFmpeg(t *testing.T) string {
	t.Helper()
	ffmpeg, _ := testutil.EnsureFfmpegBinaries(t)
	return ffmpeg
}

func newTestEngine(t *testing.T, ffmpegPath string) *Engine {
	t.Helper()
	return NewEngine(ffmpegPath, t.TempDir(), &fakeLogger{})
}
