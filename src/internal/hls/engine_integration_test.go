package hls

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuickOrBeDead/graftik-video-player/internal/data"
)

func genTestVideo(t *testing.T, ffmpegPath, outputPath string) {
	t.Helper()
	cmd := exec.Command(ffmpegPath,
		"-f", "lavfi", "-i", "testsrc=duration=5:size=320x240:rate=30",
		"-f", "lavfi", "-i", "anullsrc=r=44100:cl=stereo",
		"-c:v", "libx264", "-preset", "ultrafast", "-crf", "51",
		"-c:a", "aac", "-shortest", "-y", outputPath,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to generate test video: %v\n%s", err, out)
	}
}

func TestEngine_StartStream_ProducesHLS(t *testing.T) {
	ffmpegPath := findFFmpeg(t)
	e := newTestEngine(t, ffmpegPath)

	inputPath := filepath.Join(t.TempDir(), "test.mp4")
	genTestVideo(t, ffmpegPath, inputPath)

	streamID, err := e.StartStream(inputPath, &data.StreamInfo{
		Action: "remux",
	})
	if err != nil {
		t.Fatalf("StartStream: %v", err)
	}
	if streamID == "" {
		t.Fatal("expected non-empty stream ID")
	}

	stream, ok := e.streams[streamID]
	if !ok {
		t.Fatal("stream not registered")
	}
	if stream.Cmd == nil || stream.Cmd.Process == nil {
		t.Fatal("expected running process")
	}
	if stream.Dir == "" {
		t.Fatal("expected non-empty stream dir")
	}

	playlistPath := filepath.Join(stream.Dir, "stream.m3u8")
	deadline := time.After(10 * time.Second)
	for {
		if _, err := os.Stat(playlistPath); err == nil {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for playlist to be created")
		default:
			time.Sleep(200 * time.Millisecond)
		}
	}

	data, err := os.ReadFile(playlistPath)
	if err != nil {
		t.Fatalf("read playlist: %v", err)
	}
	if !strings.Contains(string(data), "#EXTM3U") {
		t.Fatal("playlist does not contain #EXTM3U header")
	}
	if !strings.Contains(string(data), ".ts") {
		t.Fatal("playlist does not contain any segment references")
	}
}

func TestEngine_StopStream_KillsProcessAndCleansUp(t *testing.T) {
	ffmpegPath := findFFmpeg(t)
	e := newTestEngine(t, ffmpegPath)

	inputPath := filepath.Join(t.TempDir(), "test.mp4")
	genTestVideo(t, ffmpegPath, inputPath)

	streamID, err := e.StartStream(inputPath, &data.StreamInfo{
		Action: "remux",
	})
	if err != nil {
		t.Fatalf("StartStream: %v", err)
	}

	stream := e.streams[streamID]
	streamDir := stream.Dir

	e.StopStream(streamID)

	if _, ok := e.streams[streamID]; ok {
		t.Fatal("stream should be removed after StopStream")
	}
	if _, err := os.Stat(streamDir); !os.IsNotExist(err) {
		t.Fatalf("stream dir should be removed: %v", err)
	}
}

func TestEngine_StopStream_UnknownID(t *testing.T) {
	e := newTestEngine(t, "ffmpeg")
	e.StopStream("nonexistent")
}

func TestEngine_StopStream_Idempotent(t *testing.T) {
	ffmpegPath := findFFmpeg(t)
	e := newTestEngine(t, ffmpegPath)

	inputPath := filepath.Join(t.TempDir(), "test.mp4")
	genTestVideo(t, ffmpegPath, inputPath)

	streamID, err := e.StartStream(inputPath, &data.StreamInfo{
		Action: "remux",
	})
	if err != nil {
		t.Fatalf("StartStream: %v", err)
	}

	e.StopStream(streamID)
	e.StopStream(streamID)
}

func TestEngine_Shutdown_KillsAllStreams(t *testing.T) {
	ffmpegPath := findFFmpeg(t)
	baseDir := t.TempDir()
	e := NewEngine(ffmpegPath, baseDir, &fakeLogger{})

	inputPath := filepath.Join(t.TempDir(), "test.mp4")
	genTestVideo(t, ffmpegPath, inputPath)

	ids := make([]string, 2)
	for i := range ids {
		id, err := e.StartStream(inputPath, &data.StreamInfo{
			Action: "remux",
		})
		if err != nil {
			t.Fatalf("StartStream %d: %v", i, err)
		}
		ids[i] = id
	}

	e.Shutdown()

	if len(e.streams) != 0 {
		t.Fatalf("expected 0 streams after shutdown, got %d", len(e.streams))
	}
	if _, err := os.Stat(baseDir); !os.IsNotExist(err) {
		t.Fatalf("base dir should be removed after shutdown: %v", err)
	}
}

func TestEngine_MultipleStreams_Independent(t *testing.T) {
	ffmpegPath := findFFmpeg(t)
	e := newTestEngine(t, ffmpegPath)

	inputPath := filepath.Join(t.TempDir(), "test.mp4")
	genTestVideo(t, ffmpegPath, inputPath)

	id1, err := e.StartStream(inputPath, &data.StreamInfo{Action: "remux"})
	if err != nil {
		t.Fatalf("StartStream 1: %v", err)
	}
	id2, err := e.StartStream(inputPath, &data.StreamInfo{Action: "remux"})
	if err != nil {
		t.Fatalf("StartStream 2: %v", err)
	}
	if id1 == id2 {
		t.Fatal("expected different stream IDs")
	}

	dir1 := e.streams[id1].Dir
	dir2 := e.streams[id2].Dir

	e.StopStream(id1)

	if _, err := os.Stat(dir1); !os.IsNotExist(err) {
		t.Fatal("stream 1 dir should be removed")
	}
	if _, err := os.Stat(dir2); os.IsNotExist(err) {
		t.Fatal("stream 2 dir should still exist")
	}
	if _, ok := e.streams[id2]; !ok {
		t.Fatal("stream 2 should still be registered")
	}

	e.StopStream(id2)
}

func TestEngine_StartStream_NonExistentInput_StartsThenProcessExits(t *testing.T) {
	ffmpegPath := findFFmpeg(t)
	e := newTestEngine(t, ffmpegPath)

	streamID, err := e.StartStream("/nonexistent/input.mp4", &data.StreamInfo{
		Action: "remux",
	})
	if err != nil {
		t.Fatalf("StartStream should not return error (process starts async): %v", err)
	}

	stream, ok := e.streams[streamID]
	if !ok {
		t.Fatal("stream should be registered")
	}

	err = stream.Cmd.Wait()
	if err == nil {
		t.Fatal("expected ffmpeg to exit with error for non-existent input")
	}

	e.StopStream(streamID)
}
