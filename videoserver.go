package main

import (
	"net"
	"net/http"
	"os"
	"path/filepath"

	graftikLogger "graftik-wails/internal/logger"
)

type VideoServer struct {
	mux      *http.ServeMux
	listener net.Listener
	port     int
	log      graftikLogger.Logger
}

func NewVideoServer(log graftikLogger.Logger) (*VideoServer, error) {
	if log == nil {
		panic("logger must not be nil")
	}
	vs := &VideoServer{log: log}

	vs.mux = http.NewServeMux()

	vs.mux.HandleFunc("/api/video", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Range")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		videoPath := r.URL.Query().Get("path")
		vs.log.Debug("VideoServer /api/video: video request", "method", r.Method, "path", videoPath, "remote", r.RemoteAddr)
		if videoPath == "" || !filepath.IsAbs(videoPath) {
			vs.log.Debug("VideoServer /api/video: invalid video path", "path", videoPath)
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		if _, err := os.Stat(videoPath); os.IsNotExist(err) {
			vs.log.Debug("VideoServer /api/video: video file not found", "path", videoPath)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		vs.log.Debug("VideoServer /api/video: serving video file", "path", videoPath)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Accept-Ranges", "bytes")
		http.ServeFile(w, r, videoPath)
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	vs.port = listener.Addr().(*net.TCPAddr).Port
	vs.listener = listener
	go http.Serve(listener, vs.mux)

	return vs, nil
}

func (vs *VideoServer) Port() int {
	return vs.port
}

func (vs *VideoServer) RegisterHLS(hlsDir string) {
	vs.log.Debug("VideoServer: hls http handler is registered to /hls/", "hlsDir", hlsDir)
	vs.mux.Handle("/hls/", http.StripPrefix("/hls/", http.FileServer(http.Dir(hlsDir))))
}
