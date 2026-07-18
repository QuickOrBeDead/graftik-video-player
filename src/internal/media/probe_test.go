package media

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/QuickOrBeDead/graftik-video-player/internal/testutil"
)

func findFFprobe(t *testing.T) string {
	t.Helper()
	_, ffprobe := testutil.EnsureFfmpegBinaries(t)
	return ffprobe
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	return err == nil, err
}

func TestContainerDisplayName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"mov,mp4,m4v,3gp,3g2,avci", "MP4"},
		{"mp4", "MP4"},
		{"mov", "MP4"},
		{"m4v", "MP4"},
		{"matroska", "MKV"},
		{"mpegts", "TS"},
		{"ogg", "OGG"},
		{"webm", "WebM"},
		{"flv", "FLV"},
		{"avi", "AVI"},
		{"asf", "WMV"},
		{"wmf", "WMV"},
		{"mpeg", "MPEG"},
		{"mpegvideo", "MPEG"},
		{"unknown", "UNKNOWN"},
		{"avi,mpegts", "AVI"},
		{"WEBM", "WebM"},
		{"Matroska", "MKV"},
	}
	for _, tc := range tests {
		got := containerDisplayName(tc.input)
		if got != tc.want {
			t.Errorf("containerDisplayName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestCodecDisplayName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"h264", "H.264"},
		{"avc1", "H.264"},
		{"avc", "H.264"},
		{"hevc", "HEVC"},
		{"h265", "HEVC"},
		{"vp9", "VP9"},
		{"vp8", "VP8"},
		{"av1", "AV1"},
		{"aac", "AAC"},
		{"mp3", "MP3"},
		{"vorbis", "Vorbis"},
		{"opus", "Opus"},
		{"ac3", "Dolby Digital"},
		{"eac3", "Dolby Digital"},
		{"flac", "FLAC"},
		{"mpeg4", "MPEG-4"},
		{"theora", "Theora"},
		{"wmav2", "WMA"},
		{"prores", "ProRes"},
		{"dnxhd", "DNxHD"},
		{"H264", "H.264"},
		{"VP9", "VP9"},
		{"unknown", "UNKNOWN"},
	}
	for _, tc := range tests {
		got := codecDisplayName(tc.input)
		if got != tc.want {
			t.Errorf("codecDisplayName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestIsNativeExtension(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"video.mp4", true},
		{"video.mov", true},
		{"video.m4v", true},
		{"video.3gp", true},
		{"video.3g2", true},
		{"video.webm", true},
		{"video.ogg", true},
		{"video.ogv", true},
		{"/path/to/video.MP4", true},
		{"/path/to/video.WebM", true},
		{"video.mkv", false},
		{"video.avi", false},
		{"video.flv", false},
		{"video.ts", false},
		{"video.wmv", false},
		{"video.mpeg", false},
		{"video.txt", false},
	}
	for _, tc := range tests {
		got := IsNativeExtension(tc.path)
		if got != tc.want {
			t.Errorf("IsNativeExtension(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestClassifyStream_AlwaysNative(t *testing.T) {
	p := NewProber(testutil.NopLogger{})
	tests := []struct {
		format string
	}{
		{"webm"},
		{"ogg"},
	}
	for _, tc := range tests {
		action, label := p.classifyStream(tc.format, nil)
		if action != "native" {
			t.Errorf("classifyStream(%q, ...) action = %q, want %q", tc.format, action, "native")
		}
		if label != "Direct Native" {
			t.Errorf("classifyStream(%q, ...) label = %q, want %q", tc.format, label, "Direct Native")
		}
	}
}

func TestClassifyStream_NativeContainerNativeCodec(t *testing.T) {
	p := NewProber(testutil.NopLogger{})
	tests := []struct {
		format string
		codec  string
	}{
		{"mp4", "h264"},
		{"mp4", "avc1"},
		{"mp4", "vp8"},
		{"mp4", "vp9"},
		{"mp4", "mpeg4"},
		{"mp4", "mpeg2video"},
		{"mp4", "h263"},
		{"mov", "h264"},
		{"m4v", "vp8"},
	}
	for _, tc := range tests {
		streams := []ffprobeStream{{CodecType: "video", CodecName: tc.codec}}
		action, label := p.classifyStream(tc.format, streams)
		if action != "native" {
			t.Errorf("classifyStream(%q, [{video %q}]) action = %q, want %q", tc.format, tc.codec, action, "native")
		}
		if label != "Direct Native" {
			t.Errorf("classifyStream(%q, [{video %q}]) label = %q, want %q", tc.format, tc.codec, label, "Direct Native")
		}
	}
}

func TestClassifyStream_NativeContainerHEVC(t *testing.T) {
	p := NewProber(testutil.NopLogger{})
	tests := []struct {
		format string
		codec  string
	}{
		{"mp4", "hevc"},
		{"mp4", "h265"},
		{"mov", "hevc"},
		{"m4v", "h265"},
	}
	for _, tc := range tests {
		streams := []ffprobeStream{{CodecType: "video", CodecName: tc.codec}}
		action, label := p.classifyStream(tc.format, streams)
		if action != "native" {
			t.Errorf("classifyStream(%q, [{video %q}]) action = %q, want %q", tc.format, tc.codec, action, "native")
		}
		if label != "Direct Native" {
			t.Errorf("classifyStream(%q, [{video %q}]) label = %q, want %q", tc.format, tc.codec, label, "Direct Native")
		}
	}
}

func TestClassifyStream_SWTranscode_NativeContainer(t *testing.T) {
	p := NewProber(testutil.NopLogger{})
	tests := []struct {
		format string
		codec  string
	}{
		{"mp4", "av1"},
		{"mp4", "prores"},
		{"mov", "dnxhd"},
		{"m4v", "theora"},
	}
	for _, tc := range tests {
		streams := []ffprobeStream{{CodecType: "video", CodecName: tc.codec}}
		action, label := p.classifyStream(tc.format, streams)
		if action != "sw_transcode" {
			t.Errorf("classifyStream(%q, [{video %q}]) action = %q, want %q", tc.format, tc.codec, action, "sw_transcode")
		}
		if label != "SW Transcode" {
			t.Errorf("classifyStream(%q, [{video %q}]) label = %q, want %q", tc.format, tc.codec, label, "SW Transcode")
		}
	}
}

func TestClassifyStream_Remux(t *testing.T) {
	p := NewProber(testutil.NopLogger{})
	tests := []struct {
		format string
		codec  string
	}{
		{"matroska", "h264"},
		{"avi", "h264"},
		{"avi", "avc1"},
		{"flv", "h264"},
		{"mpegts", "h264"},
		{"mpegts", "mpeg2video"},
		{"matroska", "vp8"},
		{"matroska", "vp9"},
		{"matroska", "mpeg4"},
		{"flv", "h263"},
	}
	for _, tc := range tests {
		streams := []ffprobeStream{{CodecType: "video", CodecName: tc.codec}}
		action, label := p.classifyStream(tc.format, streams)
		if action != "remux" {
			t.Errorf("classifyStream(%q, [{video %q}]) action = %q, want %q", tc.format, tc.codec, action, "remux")
		}
		if label != "Remux" {
			t.Errorf("classifyStream(%q, [{video %q}]) label = %q, want %q", tc.format, tc.codec, label, "Remux")
		}
	}
}

func TestClassifyStream_SWTranscode_NonNative(t *testing.T) {
	p := NewProber(testutil.NopLogger{})
	tests := []struct {
		format string
		codec  string
	}{
		{"matroska", "av1"},
		{"avi", "hevc"},
		{"mpegts", "av1"},
	}
	for _, tc := range tests {
		streams := []ffprobeStream{{CodecType: "video", CodecName: tc.codec}}
		action, label := p.classifyStream(tc.format, streams)
		if action != "sw_transcode" {
			t.Errorf("classifyStream(%q, [{video %q}]) action = %q, want %q", tc.format, tc.codec, action, "sw_transcode")
		}
		if label != "SW Transcode" {
			t.Errorf("classifyStream(%q, [{video %q}]) label = %q, want %q", tc.format, tc.codec, label, "SW Transcode")
		}
	}
}

func TestClassifyStream_NoVideo(t *testing.T) {
	p := NewProber(testutil.NopLogger{})
	streams := []ffprobeStream{{CodecType: "audio", CodecName: "aac"}}
	action, label := p.classifyStream("matroska", streams)
	if action != "sw_transcode" {
		t.Errorf("classifyStream with no video: action = %q, want %q", action, "sw_transcode")
	}
	if label != "SW Transcode" {
		t.Errorf("classifyStream with no video: label = %q, want %q", label, "SW Transcode")
	}
}

func TestClassifyStream_CompoundFormat(t *testing.T) {
	p := NewProber(testutil.NopLogger{})
	streams := []ffprobeStream{{CodecType: "video", CodecName: "h264"}}
	action, label := p.classifyStream("avi,mpegts", streams)
	if action != "remux" {
		t.Errorf("classifyStream(avi,mpegts, h264): action = %q, want remux", action)
	}
	if label != "Remux" {
		t.Errorf("classifyStream(avi,mpegts, h264): label = %q, want Remux", label)
	}
}

func TestClassifyStream_CaseInsensitiveFormat(t *testing.T) {
	p := NewProber(testutil.NopLogger{})
	tests := []struct {
		format       string
		wantAction   string
	}{
		{"WEBM", "native"},
		{"Matroska", "remux"},
		{"MP4", "native"},
	}
	for _, tc := range tests {
		streams := []ffprobeStream{{CodecType: "video", CodecName: "h264"}}
		action, _ := p.classifyStream(tc.format, streams)
		if action != tc.wantAction {
			t.Errorf("classifyStream(%q, h264) action = %q, want %q", tc.format, action, tc.wantAction)
		}
	}
}

func TestNewProber_NilLogger(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("NewProber(nil) did not panic")
		}
	}()
	NewProber(nil)
}

func TestNewProber(t *testing.T) {
	p := NewProber(testutil.NopLogger{})
	if p == nil {
		t.Fatal("NewProber returned nil")
	}
}

func TestProbe_BadPath(t *testing.T) {
	p := NewProber(testutil.NopLogger{})
	_, err := p.Probe("ffprobe", "/nonexistent/video.mp4")
	if err == nil {
		t.Fatal("Probe with bad path should return error")
	}
}

func TestProbe_Success(t *testing.T) {
	p := NewProber(testutil.NopLogger{})
	ffprobePath := findFFprobe(t)

	testVideo := filepath.Join("..", "..", "testdata", "sample.mp4")
	if ok, _ := pathExists(testVideo); !ok {
		t.Skipf("test video not found at %s", testVideo)
	}

	info, err := p.Probe(ffprobePath, testVideo)
	if err != nil {
		t.Fatalf("Probe returned error: %v", err)
	}
	if info == nil {
		t.Fatal("Probe returned nil info")
	}
	if info.Container == "" {
		t.Error("Container is empty")
	}
	if info.VideoCodec == "" {
		t.Error("VideoCodec is empty")
	}
	if info.Width <= 0 || info.Height <= 0 {
		t.Errorf("Dimensions invalid: %dx%d", info.Width, info.Height)
	}
}

func TestDetectHWEncoder_NoFFmpeg(t *testing.T) {
	p := NewProber(testutil.NopLogger{})
	result := p.DetectHWEncoder("/nonexistent/ffmpeg")
	if result != "" {
		t.Errorf("DetectHWEncoder with bad path = %q, want empty", result)
	}
}

func TestDetectHWEncoder(t *testing.T) {
	ffmpegPath, _ := testutil.EnsureFfmpegBinaries(t)
	p := NewProber(testutil.NopLogger{})

	result := p.DetectHWEncoder(ffmpegPath)
	accept := map[string]bool{
		"":              true,
		"h264_nvenc":    true,
		"h264_qsv":      true,
		"h264_amf":      true,
	}
	if !accept[result] {
		t.Errorf("DetectHWEncoder returned unexpected value: %q", result)
	}
}
