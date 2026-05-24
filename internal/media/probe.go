package media

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"graftik-wails/internal/data"
)

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

func Probe(ffprobePath, videoPath string) (*data.StreamInfo, error) {
	cmd := exec.Command(ffprobePath,
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

	info.Action, info.ActionLabel = classifyStream(probe.Format.FormatName, probe.Streams)

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

func classifyStream(formatName string, streams []ffprobeStream) (action, actionLabel string) {
	formatName = strings.ToLower(formatName)
	parts := strings.Split(formatName, ",")

	nativeContainers := map[string]bool{
		"mov": true, "mp4": true, "m4v": true, "3gp": true, "3g2": true,
	}

	alwaysNativeContainers := map[string]bool{
		"webm": true, "ogg": true,
	}

	for _, p := range parts {
		if alwaysNativeContainers[p] {
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
	for _, p := range parts {
		if nativeContainers[p] {
			isNativeContainer = true
			break
		}
	}

	if isNativeContainer {
		if nativeCodecs[videoCodec] {
			return "native", "Direct Native"
		}
		if videoCodec == "hevc" || videoCodec == "h265" {
			return "native", "Direct Native"
		}
		return "sw_transcode", "SW Transcode"
	}

	if nativeCodecs[videoCodec] {
		return "remux", "Remux"
	}

	return "sw_transcode", "SW Transcode"
}

func DetectHWEncoder(ffmpegPath string) string {
	cmd := exec.Command(ffmpegPath, "-encoders")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	out := string(output)

	switch {
	case strings.Contains(out, "h264_nvenc"):
		return "h264_nvenc"
	case strings.Contains(out, "h264_qsv"):
		return "h264_qsv"
	case strings.Contains(out, "h264_amf"):
		return "h264_amf"
	default:
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
