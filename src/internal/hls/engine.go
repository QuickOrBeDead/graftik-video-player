package hls

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/QuickOrBeDead/graftik-video-player/internal/command"
	"github.com/QuickOrBeDead/graftik-video-player/internal/data"
	graftikLogger "github.com/QuickOrBeDead/graftik-video-player/internal/logger"

	"github.com/google/uuid"
)

type Stream struct {
	ID  string
	Cmd *exec.Cmd
	Dir string
}

type Engine struct {
	mu         sync.Mutex
	ffmpegPath string
	baseDir    string
	streams    map[string]*Stream
	log        graftikLogger.Logger
}

func NewEngine(ffmpegPath, baseDir string, log graftikLogger.Logger) *Engine {
	if log == nil {
		panic("hls: logger is required")
	}
	log.Debug("hls: creating engine", "ffmpegPath", ffmpegPath, "baseDir", baseDir)
	return &Engine{
		ffmpegPath: ffmpegPath,
		baseDir:    baseDir,
		streams:    make(map[string]*Stream),
		log:        log,
	}
}

func (e *Engine) BaseDir() string {
	return e.baseDir
}

func (e *Engine) StartStream(path string, info *data.StreamInfo) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	streamID := uuid.New().String()
	outDir := filepath.Join(e.baseDir, streamID)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", fmt.Errorf("create stream dir: %w", err)
	}

	segPattern := filepath.Join(outDir, "seg_%05d.ts")
	playlistPath := filepath.Join(outDir, "stream.m3u8")

	args := e.buildFFmpegArgs(path, segPattern, playlistPath, info)

	e.log.Debug("hls: starting stream", "streamID", streamID, "input", path, "action", info.Action)

	cmd := command.CreateHiddenCmd(e.ffmpegPath, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		os.RemoveAll(outDir)
		return "", fmt.Errorf("start ffmpeg: %w", err)
	}

	e.streams[streamID] = &Stream{
		ID:  streamID,
		Cmd: cmd,
		Dir: outDir,
	}

	e.log.Debug("hls: stream started", "streamID", streamID)
	return streamID, nil
}

func (e *Engine) StopStream(streamID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	stream, ok := e.streams[streamID]
	if !ok {
		e.log.Debug("hls: stream not found for stopping", "streamID", streamID)
		return
	}
	e.log.Debug("hls: stopping stream", "streamID", streamID)

	if stream.Cmd != nil && stream.Cmd.Process != nil {
		if err := stream.Cmd.Process.Kill(); err != nil {
			e.log.Error("hls: failed to kill stream process", "streamID", streamID, "error", err)
		}
		if err := stream.Cmd.Wait(); err != nil {
			e.log.Error("hls: failed to wait for stream process", "streamID", streamID, "error", err)
		}
	}

	os.RemoveAll(stream.Dir)
	delete(e.streams, streamID)
}

func (e *Engine) Shutdown() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.log.Debug("hls: shutting down engine")
	for id, stream := range e.streams {
		if stream.Cmd != nil && stream.Cmd.Process != nil {
			if err := stream.Cmd.Process.Kill(); err != nil {
				e.log.Error("hls: failed to kill stream process during shutdown", "streamID", id, "error", err)
			}
			if err := stream.Cmd.Wait(); err != nil {
				e.log.Error("hls: failed to wait for stream process during shutdown", "streamID", id, "error", err)
			}
		}
		os.RemoveAll(stream.Dir)
		delete(e.streams, id)
	}

	os.RemoveAll(e.baseDir)
}

func hwEncoderFFmpegName(short string) string {
	switch short {
	case "NVENC":
		return "h264_nvenc"
	case "QSV":
		return "h264_qsv"
	case "AMF":
		return "h264_amf"
	}
	return ""
}

func (e *Engine) buildFFmpegArgs(inputPath, segPattern, playlistPath string, info *data.StreamInfo) []string {
	args := []string{"-hwaccel", "auto", "-i", inputPath}

	switch info.Action {
	case "remux":
		args = append(args, "-c", "copy")
	case "hw_transcode":
		hwName := hwEncoderFFmpegName(info.HWEncoder)
		if hwName != "" {
			args = append(args, "-c:v", hwName)
		} else {
			args = append(args, "-c:v", "libx264", "-preset", "fast", "-crf", "23")
		}
		switch info.HWEncoder {
		case "NVENC":
			args = append(args, "-preset", "p4", "-cq", "23")
		case "QSV":
			args = append(args, "-global_quality", "23")
		case "AMF":
			args = append(args, "-quality", "balanced", "-usage", "transcoding")
		}
		args = append(args, "-c:a", "aac", "-b:a", "128k")
	case "sw_transcode":
		args = append(args, "-c:v", "libx264", "-preset", "fast", "-crf", "23")
		args = append(args, "-c:a", "aac", "-b:a", "128k")
	default:
		args = append(args, "-c", "copy")
	}

	args = append(args,
		"-f", "hls",
		"-hls_time", "6",
		"-hls_list_size", "0",
		"-hls_segment_filename", segPattern,
		playlistPath,
	)

	return args
}
