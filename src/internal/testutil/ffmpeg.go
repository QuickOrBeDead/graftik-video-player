package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func EnsureFfmpegBinaries(t *testing.T) (ffmpeg, ffprobe string) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := filepath.Join("..", "..", "build", runtime.GOOS, "bin")

	ffmpegCandidates := []string{"ffmpeg", "ffmpeg.exe"}
	ffprobeCandidates := []string{"ffprobe", "ffprobe.exe"}

	for _, name := range ffmpegCandidates {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			ffmpeg = p
			break
		}
	}
	for _, name := range ffprobeCandidates {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			ffprobe = p
			break
		}
	}

	if ffmpeg != "" && ffprobe != "" {
		return ffmpeg, ffprobe
	}

	script := filepath.Join("..", "..", "build", runtime.GOOS, "download-ffmpeg"+ScriptExt())
	cmd := ExecCommand(script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("download-ffmpeg failed: %v\n%s", err, out)
	}

	for _, name := range ffmpegCandidates {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			ffmpeg = p
			break
		}
	}
	for _, name := range ffprobeCandidates {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			ffprobe = p
			break
		}
	}

	if ffmpeg == "" {
		t.Fatalf("ffmpeg not found after running download script at %s", dir)
	}
	if ffprobe == "" {
		t.Fatalf("ffprobe not found after running download script at %s", dir)
	}

	return ffmpeg, ffprobe
}
