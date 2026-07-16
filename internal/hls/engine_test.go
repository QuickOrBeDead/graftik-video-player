package hls

import (
	"testing"

	"github.com/QuickOrBeDead/graftik-video-player/internal/data"
)

func TestNewEnginePanicsOnNilLogger(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil logger")
		}
	}()
	NewEngine("ffmpeg", "/tmp", nil)
}

func TestNewEngineAndBaseDir(t *testing.T) {
	e := NewEngine("ffmpeg", "/tmp/hls", &fakeLogger{})
	if e.BaseDir() != "/tmp/hls" {
		t.Fatalf("expected /tmp/hls, got %s", e.BaseDir())
	}
}

func TestHwEncoderFFmpegName(t *testing.T) {
	tests := []struct {
		short  string
		expect string
	}{
		{"NVENC", "h264_nvenc"},
		{"QSV", "h264_qsv"},
		{"AMF", "h264_amf"},
		{"", ""},
		{"unknown", ""},
		{"nvenc", ""},
	}
	for _, tc := range tests {
		got := hwEncoderFFmpegName(tc.short)
		if got != tc.expect {
			t.Errorf("hwEncoderFFmpegName(%q) = %q, want %q", tc.short, got, tc.expect)
		}
	}
}

func TestBuildFFmpegArgs_remux(t *testing.T) {
	e := NewEngine("ffmpeg", "/tmp", &fakeLogger{})
	args := e.buildFFmpegArgs("/input/video.mkv", "/out/seg_%05d.ts", "/out/stream.m3u8", &data.StreamInfo{
		Action: "remux",
	})
	if len(args) == 0 {
		t.Fatal("expected non-empty args")
	}
	if args[0] != "-hwaccel" {
		t.Fatalf("expected first arg -hwaccel, got %s", args[0])
	}
	if args[2] != "-i" {
		t.Fatalf("expected arg[2] -i, got %s", args[2])
	}
	if args[3] != "/input/video.mkv" {
		t.Fatalf("expected input path, got %s", args[3])
	}
	if args[4] != "-c" || args[5] != "copy" {
		t.Fatalf("expected -c copy after input, got %s %s", args[4], args[5])
	}
	assertHLSSuffix(t, args, "/out/stream.m3u8")
}

func TestBuildFFmpegArgs_sw_transcode(t *testing.T) {
	e := NewEngine("ffmpeg", "/tmp", &fakeLogger{})
	args := e.buildFFmpegArgs("/input/video.mkv", "/out/seg_%05d.ts", "/out/stream.m3u8", &data.StreamInfo{
		Action: "sw_transcode",
	})
	expectArgs := []string{"-c:v", "libx264", "-preset", "fast", "-crf", "23"}
	for i, v := range expectArgs {
		if args[4+i] != v {
			t.Fatalf("sw_transcode: args[%d] = %q, want %q", 4+i, args[4+i], v)
		}
	}
	assertHLSSuffix(t, args, "/out/stream.m3u8")
}

func TestBuildFFmpegArgs_hw_transcode_NVENC(t *testing.T) {
	e := NewEngine("ffmpeg", "/tmp", &fakeLogger{})
	args := e.buildFFmpegArgs("/input/video.mkv", "/out/seg_%05d.ts", "/out/stream.m3u8", &data.StreamInfo{
		Action:    "hw_transcode",
		HWEncoder: "NVENC",
	})
	expectSub := []string{"-c:v", "h264_nvenc", "-preset", "p4", "-cq", "23"}
	for i, v := range expectSub {
		if args[4+i] != v {
			t.Fatalf("NVENC: args[%d] = %q, want %q", 4+i, args[4+i], v)
		}
	}
	assertHLSSuffix(t, args, "/out/stream.m3u8")
}

func TestBuildFFmpegArgs_hw_transcode_QSV(t *testing.T) {
	e := NewEngine("ffmpeg", "/tmp", &fakeLogger{})
	args := e.buildFFmpegArgs("/input/video.mkv", "/out/seg_%05d.ts", "/out/stream.m3u8", &data.StreamInfo{
		Action:    "hw_transcode",
		HWEncoder: "QSV",
	})
	expectSub := []string{"-c:v", "h264_qsv", "-global_quality", "23"}
	for i, v := range expectSub {
		if args[4+i] != v {
			t.Fatalf("QSV: args[%d] = %q, want %q", 4+i, args[4+i], v)
		}
	}
	assertHLSSuffix(t, args, "/out/stream.m3u8")
}

func TestBuildFFmpegArgs_hw_transcode_AMF(t *testing.T) {
	e := NewEngine("ffmpeg", "/tmp", &fakeLogger{})
	args := e.buildFFmpegArgs("/input/video.mkv", "/out/seg_%05d.ts", "/out/stream.m3u8", &data.StreamInfo{
		Action:    "hw_transcode",
		HWEncoder: "AMF",
	})
	expectSub := []string{"-c:v", "h264_amf", "-quality", "balanced", "-usage", "transcoding"}
	for i, v := range expectSub {
		if args[4+i] != v {
			t.Fatalf("AMF: args[%d] = %q, want %q", 4+i, args[4+i], v)
		}
	}
	assertHLSSuffix(t, args, "/out/stream.m3u8")
}

func TestBuildFFmpegArgs_hw_transcode_unknown_encoder_falls_back_to_software(t *testing.T) {
	e := NewEngine("ffmpeg", "/tmp", &fakeLogger{})
	args := e.buildFFmpegArgs("/input/video.mkv", "/out/seg_%05d.ts", "/out/stream.m3u8", &data.StreamInfo{
		Action:    "hw_transcode",
		HWEncoder: "",
	})
	expectSub := []string{"-c:v", "libx264", "-preset", "fast", "-crf", "23"}
	for i, v := range expectSub {
		if args[4+i] != v {
			t.Fatalf("fallback: args[%d] = %q, want %q", 4+i, args[4+i], v)
		}
	}
	assertHLSSuffix(t, args, "/out/stream.m3u8")
}

func TestBuildFFmpegArgs_default_action_is_copy(t *testing.T) {
	e := NewEngine("ffmpeg", "/tmp", &fakeLogger{})
	args := e.buildFFmpegArgs("/input/video.mkv", "/out/seg_%05d.ts", "/out/stream.m3u8", &data.StreamInfo{
		Action: "unknown_action",
	})
	if args[4] != "-c" || args[5] != "copy" {
		t.Fatalf("expected -c copy for unknown action, got %s %s", args[4], args[5])
	}
	assertHLSSuffix(t, args, "/out/stream.m3u8")
}

func assertHLSSuffix(t *testing.T, args []string, playlistPath string) {
	t.Helper()
	n := len(args)
	if n < 9 {
		t.Fatalf("too few args: %d", n)
	}
	if args[n-9] != "-f" || args[n-8] != "hls" {
		t.Errorf("missing -f hls: %v", args[n-9:n-7])
	}
	if args[n-7] != "-hls_time" || args[n-6] != "6" {
		t.Errorf("missing hls_time: %v", args[n-7:n-5])
	}
	if args[n-5] != "-hls_list_size" || args[n-4] != "0" {
		t.Errorf("missing hls_list_size: %v", args[n-5:n-3])
	}
	if args[n-3] != "-hls_segment_filename" {
		t.Errorf("missing -hls_segment_filename at args[%d]", n-3)
	}
	if args[n-1] != playlistPath {
		t.Errorf("expected playlist path %q, got %q", playlistPath, args[n-1])
	}
}
