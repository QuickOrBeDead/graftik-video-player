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

	"github.com/Masterminds/semver/v3"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var appVersion = "0.0.0"
var releaseYear = "" // set via ldflags to freeze at build time

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
	Draft       bool          `json:"draft"`
	Assets      []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func (a *App) GetAppVersion() string {
	return appVersion
}

func (a *App) GetReleaseYear() string {
	if releaseYear != "" {
		return releaseYear
	}
	return strconv.Itoa(time.Now().Year())
}

func (a *App) CheckForUpdates() (*UpdateInfo, error) {
	var includePrerelease bool
	if a.store != nil {
		prefs := a.store.GetPreferences()
		includePrerelease = prefs.IncludePrereleasesForUpdates
	}

	if includePrerelease {
		return a.checkForUpdatesPrerelease()
	}
	return a.checkForUpdatesStable()
}

func (a *App) checkForUpdatesStable() (*UpdateInfo, error) {
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

	return a.releaseToUpdateInfo(&release)
}

func (a *App) checkForUpdatesPrerelease() (*UpdateInfo, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	ext := ".deb"
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}

	for page := 1; ; page++ {
		url := fmt.Sprintf("https://api.github.com/repos/QuickOrBeDead/graftik-video-player/releases?per_page=100&page=%d", page)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
		}

		var releases []githubRelease
		if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		for _, release := range releases {
			if release.Draft {
				continue
			}
			for _, asset := range release.Assets {
				if strings.HasSuffix(asset.Name, ext) {
					info := a.releaseToUpdateInfoOrNil(&release)
					if info != nil {
						return info, nil
					}
					break
				}
			}
		}

		nextURL := parseNextLink(resp.Header.Get("Link"))
		if nextURL == "" {
			break
		}
	}

	return nil, nil
}

func parseNextLink(link string) string {
	for _, part := range strings.Split(link, ",") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, `rel="next"`) {
			if start := strings.Index(part, "<"); start != -1 {
				if end := strings.Index(part[start:], ">"); end != -1 {
					return part[start+1 : start+end]
				}
			}
		}
	}
	return ""
}

func (a *App) releaseToUpdateInfo(release *githubRelease) (*UpdateInfo, error) {
	info := a.releaseToUpdateInfoOrNil(release)
	if info != nil {
		return info, nil
	}
	return nil, nil
}

func (a *App) releaseToUpdateInfoOrNil(release *githubRelease) *UpdateInfo {
	if appVersion == "0.0.0" {
		return nil
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")

	cmp := semverCompare(latestVersion, appVersion)
	if cmp <= 0 {
		return nil
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

	if downloadURL == "" {
		return nil
	}

	return &UpdateInfo{
		HasUpdate:     true,
		LatestVersion: latestVersion,
		DownloadURL:   downloadURL,
		ReleaseNotes:  release.Body,
	}
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
	va, err := semver.NewVersion(a)
	if err != nil {
		return 0
	}
	vb, err := semver.NewVersion(b)
	if err != nil {
		return 0
	}
	return va.Compare(vb)
}
