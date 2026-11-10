// Package engram installs and configures the Engram persistent-memory server.
package engram

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	repoAPI    = "https://api.github.com/repos/Gentleman-Programming/engram/releases/latest"
	userAgent  = "memo-ai-installer"
	httpTimout = 10 * time.Minute
)

// Release describes the subset of the GitHub release payload we need.
type Release struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// Install downloads the latest Engram release, verifies its checksum and
// installs the binary under ~/.local/bin. It returns the installed path.
func Install(ctx context.Context, progress io.Writer) (string, error) {
	client := &http.Client{Timeout: httpTimout}
	release, err := fetchRelease(ctx, client)
	if err != nil {
		return "", err
	}
	version := strings.TrimPrefix(release.TagName, "v")
	assetName, archiveType := assetFor(version, runtime.GOOS, runtime.GOARCH)
	downloadURL, checksumURL := "", ""
	for _, asset := range release.Assets {
		switch asset.Name {
		case assetName:
			downloadURL = asset.BrowserDownloadURL
		case "checksums.txt":
			checksumURL = asset.BrowserDownloadURL
		}
	}
	if downloadURL == "" {
		return "", fmt.Errorf("release %s has no asset %s", release.TagName, assetName)
	}
	if checksumURL == "" {
		return "", errors.New("release has no checksums.txt; refusing to install unverified binary")
	}

	if progress != nil {
		fmt.Fprintf(progress, "  → %s\n", downloadURL)
	}
	archive, err := download(ctx, client, downloadURL)
	if err != nil {
		return "", err
	}
	defer os.Remove(archive)

	want, err := checksumFor(ctx, client, checksumURL, assetName)
	if err != nil {
		return "", err
	}
	got, err := fileSHA256(archive)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(want, got) {
		return "", fmt.Errorf("checksum mismatch for %s: want %s, got %s", assetName, want, got)
	}

	bin, err := extractBinary(archive, archiveType)
	if err != nil {
		return "", err
	}
	defer os.Remove(bin)

	dest, err := installPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	if err := copyExecutable(bin, dest); err != nil {
		return "", err
	}
	return dest, nil
}

func installPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	name := "engram"
	if runtime.GOOS == "windows" {
		name = "engram.exe"
	}
	return filepath.Join(home, ".local", "bin", name), nil
}

// Commands returns common locations to search for Engram.
func Commands() []string {
	home, _ := os.UserHomeDir()
	name := "engram"
	if runtime.GOOS == "windows" {
		name = "engram.exe"
	}
	return []string{
		filepath.Join(home, ".local", "bin", name),
		filepath.Join(home, "go", "bin", name),
		filepath.Join(home, "bin", name),
		"/usr/local/bin/engram",
		"/opt/homebrew/bin/engram",
	}
}

// MCPArgs are the arguments omp passes to the Engram binary to start its
// stdio MCP server.
func MCPArgs() []any {
	return []any{"mcp", "--tools=agent"}
}

func fetchRelease(ctx context.Context, client *http.Client) (Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, repoAPI, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("GitHub releases API returned %s", resp.Status)
	}
	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return Release{}, err
	}
	if release.TagName == "" {
		return Release{}, errors.New("release payload has no tag_name")
	}
	return release, nil
}

func assetFor(version, goos, goarch string) (name, archiveType string) {
	switch goos {
	case "windows":
		return fmt.Sprintf("engram_%s_%s_%s.zip", version, goos, goarch), "zip"
	default:
		return fmt.Sprintf("engram_%s_%s_%s.tar.gz", version, goos, goarch), "tar.gz"
	}
}

func download(ctx context.Context, client *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: %s", url, resp.Status)
	}
	tmp, err := os.CreateTemp("", "memo-engram-*")
	if err != nil {
		return "", err
	}
	name := tmp.Name()
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(name)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}

func checksumFor(ctx context.Context, client *http.Client, url, asset string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download checksums: %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.TrimPrefix(fields[1], "*") == asset {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("no checksum entry for %s", asset)
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func extractBinary(archive, archiveType string) (string, error) {
	switch archiveType {
	case "zip":
		return extractZip(archive)
	default:
		return extractTarGz(archive)
	}
}

func extractTarGz(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		if !isBinaryName(header.Name) {
			continue
		}
		return writeTemp(tr)
	}
	return "", errors.New("archive does not contain an engram binary")
}

func extractZip(path string) (string, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer zr.Close()
	for _, file := range zr.File {
		if file.FileInfo().IsDir() || !isBinaryName(file.Name) {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return "", err
		}
		defer rc.Close()
		return writeTemp(rc)
	}
	return "", errors.New("archive does not contain an engram binary")
}

func isBinaryName(name string) bool {
	base := filepath.Base(name)
	return base == "engram" || base == "engram.exe"
}

func writeTemp(r io.Reader) (string, error) {
	tmp, err := os.CreateTemp("", "memo-engram-bin-*")
	if err != nil {
		return "", err
	}
	name := tmp.Name()
	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		os.Remove(name)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}

func copyExecutable(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".engram-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := io.Copy(tmp, in); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(name, 0o755); err != nil {
		return err
	}
	return os.Rename(name, dst)
}
