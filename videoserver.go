package main

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
)

type VideoServer struct {
	mux      *http.ServeMux
	listener net.Listener
	port     int
}

func NewVideoServer() (*VideoServer, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/video", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Range")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		videoPath := r.URL.Query().Get("path")
		if videoPath == "" || !filepath.IsAbs(videoPath) {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		if _, err := os.Stat(videoPath); os.IsNotExist(err) {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Accept-Ranges", "bytes")
		http.ServeFile(w, r, videoPath)
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	port := listener.Addr().(*net.TCPAddr).Port
	go http.Serve(listener, mux)

	return &VideoServer{mux: mux, listener: listener, port: port}, nil
}

func (vs *VideoServer) Port() int {
	return vs.port
}

func (vs *VideoServer) RegisterHLS(hlsDir string) {
	vs.mux.Handle("/hls/", http.StripPrefix("/hls/", http.FileServer(http.Dir(hlsDir))))
}
