package media

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/QuickOrBeDead/graftik-video-player/internal/command"
	"github.com/QuickOrBeDead/graftik-video-player/internal/data"
	graftikLogger "github.com/QuickOrBeDead/graftik-video-player/internal/logger"
)

type Prober struct {
	log graftikLogger.Logger
}

func NewProber(log graftikLogger.Logger) *Prober {
	if log == nil {
		panic("media: logger is required")
	}
	return &Prober{log: log}
}

type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

type ffprobeStream struct {
	CodecType string `json:"codec_type"`
	CodecName string `json:"codec_name"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

type ffprobeFormat struct {
	FormatName string `json:"format_name"`
	Duration   string `json:"duration"`
}

func (p *Prober) Probe(ffprobePath, videoPath string) (*data.StreamInfo, error) {
	p.log.Debug("media: probing file", "ffprobePath", ffprobePath, "videoPath", videoPath)

	cmd := command.CreateHiddenCmd(ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		videoPath,
	)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe: %w", err)
	}

	var probe ffprobeOutput
	if err := json.Unmarshal(output, &probe); err != nil {
		return nil, fmt.Errorf("parse ffprobe: %w", err)
	}

	info := &data.StreamInfo{
		Container: containerDisplayName(probe.Format.FormatName),
	}

	for _, s := range probe.Streams {
		switch s.CodecType {
		case "video":
			if info.VideoCodec == "" {
				info.VideoCodec = codecDisplayName(s.CodecName)
				info.Width = s.Width
				info.Height = s.Height
			}
		case "audio":
			if info.AudioCodec == "" {
				info.AudioCodec = codecDisplayName(s.CodecName)
			}
		}
	}

	info.Action, info.ActionLabel = p.classifyStream(probe.Format.FormatName, probe.Streams)

	p.log.Debug("media: probe result",
		"videoPath", videoPath,
		"container", info.Container,
		"videoCodec", info.VideoCodec,
		"audioCodec", info.AudioCodec,
		"width", info.Width,
		"height", info.Height,
		"action", info.Action,
		"actionLabel", info.ActionLabel,
	)

	return info, nil
}

func containerDisplayName(name string) string {
	primary := strings.Split(strings.ToLower(name), ",")[0]
	switch primary {
	case "mov", "mp4", "m4v":
		return "MP4"
	case "matroska":
		return "MKV"
	case "mpegts":
		return "TS"
	case "ogg":
		return "OGG"
	case "webm":
		return "WebM"
	case "flv":
		return "FLV"
	case "avi":
		return "AVI"
	case "asf", "wmf":
		return "WMV"
	case "mpeg", "mpegvideo":
		return "MPEG"
	default:
		return strings.ToUpper(primary)
	}
}

func codecDisplayName(name string) string {
	switch strings.ToLower(name) {
	case "h264", "avc1", "avc":
		return "H.264"
	case "hevc", "h265":
		return "HEVC"
	case "vp9":
		return "VP9"
	case "vp8":
		return "VP8"
	case "av1":
		return "AV1"
	case "aac":
		return "AAC"
	case "mp3":
		return "MP3"
	case "vorbis":
		return "Vorbis"
	case "opus":
		return "Opus"
	case "ac3", "eac3":
		return "Dolby Digital"
	case "flac":
		return "FLAC"
	case "mpeg4":
		return "MPEG-4"
	case "theora":
		return "Theora"
	case "wmav2":
		return "WMA"
	case "prores":
		return "ProRes"
	case "dnxhd":
		return "DNxHD"
	default:
		return strings.ToUpper(name)
	}
}

func (p *Prober) classifyStream(formatName string, streams []ffprobeStream) (action, actionLabel string) {
	formatName = strings.ToLower(formatName)
	parts := strings.Split(formatName, ",")

	nativeContainers := map[string]bool{
		"mov": true, "mp4": true, "m4v": true, "3gp": true, "3g2": true,
	}

	alwaysNativeContainers := map[string]bool{
		"webm": true, "ogg": true,
	}

	for _, part := range parts {
		if alwaysNativeContainers[part] {
			p.log.Debug("media: classified as native (always-native container)", "format", part)
			return "native", "Direct Native"
		}
	}

	var videoCodec string
	for _, s := range streams {
		if s.CodecType == "video" {
			videoCodec = strings.ToLower(s.CodecName)
			break
		}
	}

	nativeCodecs := map[string]bool{
		"h264": true, "avc1": true, "h263": true, "mpeg4": true,
		"mpeg2video": true, "vp8": true, "vp9": true,
	}

	isNativeContainer := false
	for _, part := range parts {
		if nativeContainers[part] {
			isNativeContainer = true
			break
		}
	}

	if isNativeContainer {
		if nativeCodecs[videoCodec] {
			p.log.Debug("media: classified as native (native container + native codec)", "codec", videoCodec)
			return "native", "Direct Native"
		}
		if videoCodec == "hevc" || videoCodec == "h265" {
			p.log.Debug("media: classified as native (native container + hevc)", "codec", videoCodec)
			return "native", "Direct Native"
		}
		p.log.Debug("media: classified as sw_transcode (native container + non-native codec)", "codec", videoCodec)
		return "sw_transcode", "SW Transcode"
	}

	if nativeCodecs[videoCodec] {
		p.log.Debug("media: classified as remux (non-native container + native codec)", "codec", videoCodec)
		return "remux", "Remux"
	}

	p.log.Debug("media: classified as sw_transcode (non-native container + non-native codec)", "codec", videoCodec)
	return "sw_transcode", "SW Transcode"
}

func (p *Prober) DetectHWEncoder(ffmpegPath string) string {
	p.log.Debug("media: detecting hardware encoder", "ffmpegPath", ffmpegPath)
	cmd := command.CreateHiddenCmd(ffmpegPath, "-encoders")
	output, err := cmd.Output()
	if err != nil {
		p.log.Debug("media: failed to run ffmpeg -encoders", "error", err)
		return ""
	}
	out := string(output)

	switch {
	case strings.Contains(out, "h264_nvenc"):
		p.log.Debug("media: detected hw encoder", "encoder", "NVENC")
		return "h264_nvenc"
	case strings.Contains(out, "h264_qsv"):
		p.log.Debug("media: detected hw encoder", "encoder", "QSV")
		return "h264_qsv"
	case strings.Contains(out, "h264_amf"):
		p.log.Debug("media: detected hw encoder", "encoder", "AMF")
		return "h264_amf"
	default:
		p.log.Debug("media: no compatible hardware encoder found")
		return ""
	}
}

func IsNativeExtension(path string) bool {
	ext := strings.ToLower(path[strings.LastIndex(path, "."):])
	switch ext {
	case ".mp4", ".mov", ".m4v", ".3gp", ".3g2", ".webm", ".ogg", ".ogv":
		return true
	}
	return false
}
