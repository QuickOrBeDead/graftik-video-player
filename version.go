package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var appVersion = "0.0.0"

type UpdateInfo struct {
	HasUpdate     bool   `json:"hasUpdate"`
	LatestVersion string `json:"latestVersion"`
	DownloadURL   string `json:"downloadUrl"`
	ReleaseNotes  string `json:"releaseNotes"`
}

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Body        string        `json:"body"`
	PublishedAt string        `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func (a *App) GetAppVersion() string {
	return appVersion
}

func (a *App) CheckForUpdates() (*UpdateInfo, error) {
	url := "https://api.github.com/repos/QuickOrBeDead/graftik-video-player/releases/latest"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if a.updateETag != "" {
		req.Header.Set("If-None-Match", a.updateETag)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return nil, nil
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	a.updateETag = resp.Header.Get("ETag")

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")

	cmp := semverCompare(latestVersion, appVersion)
	if cmp <= 0 {
		return nil, nil
	}

	var ext string
	if runtime.GOOS == "windows" {
		ext = ".exe"
	} else {
		ext = ".deb"
	}

	var downloadURL string
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, ext) {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	return &UpdateInfo{
		HasUpdate:     true,
		LatestVersion: latestVersion,
		DownloadURL:   downloadURL,
		ReleaseNotes:  release.Body,
	}, nil
}

func (a *App) DownloadUpdate(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	ext := ".deb"
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}

	tmpFile := filepath.Join(os.TempDir(), "graftik-update"+ext)
	f, err := os.Create(tmpFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	pr := &progressReader{
		reader: resp.Body,
		total:  resp.ContentLength,
		onProgress: func(pct int) {
			if a.ctx != nil {
				wailsRuntime.EventsEmit(a.ctx, "update-download-progress", fmt.Sprintf(`{"percent":%d}`, pct))
			}
		},
	}

	if _, err := io.Copy(f, pr); err != nil {
		return "", err
	}

	return tmpFile, nil
}

func (a *App) InstallUpdate(path string) error {
	switch runtime.GOOS {
	case "linux":
		if _, err := exec.LookPath("pkexec"); err != nil {
			return fmt.Errorf("pkexec not found; install manually: sudo dpkg -i %s", path)
		}
		cmd := exec.Command("pkexec", "dpkg", "-i", path)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("install failed: %s: %w", string(output), err)
		}
		return nil
	case "windows":
		cmd := exec.Command(path, "/S")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("installer failed: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func semverCompare(a, b string) int {
	parse := func(v string) []int {
		parts := strings.Split(v, ".")
		nums := make([]int, 3)
		for i, p := range parts {
			n, _ := strconv.Atoi(strings.TrimSpace(p))
			if i < 3 {
				nums[i] = n
			}
		}
		return nums
	}

	va, vb := parse(a), parse(b)
	for i := 0; i < 3; i++ {
		if va[i] > vb[i] {
			return 1
		}
		if va[i] < vb[i] {
			return -1
		}
	}
	return 0
}
