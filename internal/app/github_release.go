package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ghRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Body    string `json:"body"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
	} `json:"assets"`
}

type scoredAsset struct {
	Name  string
	URL   string
	Score int
	Size  int64
}

func InstallGitHubRelease(ownerRepo string, opts Options) RunResult {
	res := RunResult{DryRun: opts.DryRun, OK: true}
	_ = ensureDirs()
	var out bytes.Buffer
	fmt.Fprintf(&out, "GitHub: %s\n", ownerRepo)
	if !ownerRepoRE.MatchString(ownerRepo) {
		res.OK = false
		res.Errors = append(res.Errors, "expected owner/repo")
		res.Output = out.String()
		return res
	}
	asset, err := latestGitHubAsset(ownerRepo, Detect())
	if err != nil {
		fmt.Fprintf(&out, "release asset not selected: %v\n", err)
		if !opts.AllowUnknown {
			fmt.Fprintf(&out, "source-build fallback is blocked by default; rerun with --allow-unknown only if you trust this repository.\n")
			res.OK = false
			res.ExitCode = 2
			res.Errors = append(res.Errors, "compatible release asset not found")
			res.Output = out.String()
			return res
		}
		fmt.Fprintf(&out, "fallback allowed: clone source and run detected build recipe\n")
		plan := BuildPlan([]Task{{Kind: "git", Items: []string{ownerRepo}}}, opts)
		r := RunPlan(plan, opts.DryRun)
		out.WriteString(r.Output)
		r.Output = out.String()
		return r
	}
	fmt.Fprintf(&out, "selected asset: %s\n", asset.Name)
	dir := filepath.Join(cacheDir(), "downloads", sanitizeName(ownerRepo))
	path := filepath.Join(dir, asset.Name)
	if opts.DryRun {
		fmt.Fprintf(&out, "would download: %s\n", asset.URL)
		fmt.Fprintf(&out, "would save to: %s\n", path)
		plan := BuildPlan([]Task{{Kind: "local", Items: []string{path}}}, Options{Yes: opts.Yes, DryRun: true, NoSecurity: true, AllowUnknown: opts.AllowUnknown, VirusTotalKey: opts.VirusTotalKey, VirusTotalUpload: opts.VirusTotalUpload})
		for i, c := range plan.Commands {
			fmt.Fprintf(&out, "[%d/%d] %s\n%s\n", i+1, len(plan.Commands), c.Title, commandLine(c))
		}
		for _, w := range plan.Warnings {
			fmt.Fprintf(&out, "warning: %s\n", w)
		}
		res.Output = out.String()
		return res
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		res.OK = false
		res.Errors = append(res.Errors, err.Error())
		res.Output = out.String()
		return res
	}
	fmt.Fprintf(&out, "downloading to: %s\n", path)
	if err := downloadFile(asset.URL, path); err != nil {
		res.OK = false
		res.Errors = append(res.Errors, err.Error())
		res.Output = out.String()
		return res
	}
	rep := ScanFile(path, SecurityOptions{VirusTotalKey: opts.VirusTotalKey, VirusTotalUpload: opts.VirusTotalUpload, AllowUnknown: opts.AllowUnknown})
	writeSecurityHuman(&out, rep)
	if !SecurityAllowsInstall(rep, opts.AllowUnknown) {
		res.OK = false
		res.ExitCode = 2
		res.Errors = append(res.Errors, "security check blocked installation")
		res.Output = out.String()
		return res
	}
	plan := BuildPlan([]Task{{Kind: "local", Items: []string{path}}}, Options{Yes: opts.Yes, DryRun: false, NoSecurity: true})
	r := RunPlan(plan, false)
	out.WriteString(r.Output)
	r.Output = out.String()
	return r
}

func latestGitHubAsset(ownerRepo string, sys SystemInfo) (scoredAsset, error) {
	releases, err := fetchGitHubReleases(ownerRepo)
	if err != nil {
		return scoredAsset{}, err
	}
	candidates := make([]scoredAsset, 0)
	for _, rel := range releases {
		for _, a := range rel.Assets {
			s := scoreAssetForSystem(a.Name, a.BrowserDownloadURL, sys)
			if s > 0 {
				candidates = append(candidates, scoredAsset{Name: a.Name, URL: a.BrowserDownloadURL, Score: s, Size: a.Size})
			}
		}
		if len(candidates) > 0 {
			break
		}
	}
	if len(candidates) == 0 {
		return scoredAsset{}, fmt.Errorf("no compatible binary asset in recent GitHub releases")
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].Size > candidates[j].Size
		}
		return candidates[i].Score > candidates[j].Score
	})
	return candidates[0], nil
}

func fetchGitHubReleases(ownerRepo string) ([]ghRelease, error) {
	urls := []string{
		"https://api.github.com/repos/" + ownerRepo + "/releases/latest",
		"https://api.github.com/repos/" + ownerRepo + "/releases?per_page=10",
	}
	var lastErr error
	for i, u := range urls {
		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("User-Agent", "Instally-Go")
		if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		client := &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("too many redirects")
				}
				h := strings.ToLower(req.URL.Hostname())
				if !strings.HasSuffix(h, "github.com") && !strings.HasSuffix(h, "githubusercontent.com") {
					return fmt.Errorf("redirect blocked: not a GitHub host: %s", h)
				}
				return nil
			},
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("GitHub API returned %s", resp.Status)
			continue
		}
		if i == 0 {
			var rel ghRelease
			if err := json.Unmarshal(body, &rel); err != nil {
				lastErr = err
				continue
			}
			return []ghRelease{rel}, nil
		}
		var rels []ghRelease
		if err := json.Unmarshal(body, &rels); err != nil {
			lastErr = err
			continue
		}
		if len(rels) > 0 {
			return rels, nil
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("no GitHub releases found")
}

func scoreAsset(name, rawurl string, sys SystemInfo) int {
	return scoreAssetForSystem(name, rawurl, sys)
}

func scoreAssetForSystem(name, rawurl string, sys SystemInfo) int {
	v := strings.ToLower(name + " " + rawurl)
	bad := []string{".sha256", ".sha512", ".sig", ".asc", ".pem", ".txt", ".json", ".yml", ".yaml", ".blockmap", ".pdb", ".debug", "checksums", "checksum", "source-code", "source_code", "symbols", "sbom", "license"}
	for _, b := range bad {
		if strings.Contains(v, b) {
			return 0
		}
	}
	if wrongArchForSystem(v, sys.Arch) {
		return 0
	}
	score := 0
	switch sys.Family {
	case Linux:
		if strings.Contains(v, "windows") || strings.Contains(v, "win32") || strings.Contains(v, "win64") || strings.Contains(v, "darwin") || strings.Contains(v, "macos") || strings.Contains(v, "osx") {
			return 0
		}
		if strings.Contains(v, "linux") {
			score += 40
		}
		if strings.Contains(v, "appimage") {
			score += 140
		}
		switch sys.Manager.ID {
		case "apt":
			if strings.HasSuffix(v, ".deb") {
				score += 130
			}
		case "dnf", "zypper":
			if strings.HasSuffix(v, ".rpm") {
				score += 130
			}
		case "pacman":
			if strings.Contains(v, ".pkg.tar.") {
				score += 130
			}
		}
		if strings.HasSuffix(v, ".tar.gz") || strings.HasSuffix(v, ".tgz") || strings.HasSuffix(v, ".tar.xz") || strings.HasSuffix(v, ".tar.zst") || strings.HasSuffix(v, ".zip") || strings.HasSuffix(v, ".7z") {
			score += 35
		}
		if strings.Contains(v, "gnu") || strings.Contains(v, "glibc") || strings.Contains(v, "static") {
			score += 12
		}
		if strings.Contains(v, "musl") {
			score -= 6
		}
	case Windows:
		if strings.Contains(v, "linux") || strings.Contains(v, "darwin") || strings.Contains(v, "macos") || strings.Contains(v, "osx") {
			return 0
		}
		if strings.Contains(v, "windows") || strings.Contains(v, "win64") || strings.Contains(v, "win32") {
			score += 50
		}
		if strings.HasSuffix(v, ".msi") {
			score += 140
		}
		if strings.HasSuffix(v, ".exe") {
			score += 120
		}
		if strings.HasSuffix(v, ".msix") || strings.HasSuffix(v, ".appx") || strings.HasSuffix(v, ".msixbundle") {
			score += 100
		}
		if strings.HasSuffix(v, ".zip") || strings.HasSuffix(v, ".7z") {
			score += 30
		}
		if strings.Contains(v, "portable") {
			score += 10
		}
	case Darwin:
		if strings.Contains(v, "linux") || strings.Contains(v, "windows") || strings.Contains(v, "win32") || strings.Contains(v, "win64") {
			return 0
		}
		if strings.Contains(v, "darwin") || strings.Contains(v, "macos") || strings.Contains(v, "osx") || strings.Contains(v, "mac") {
			score += 50
		}
		if strings.HasSuffix(v, ".dmg") {
			score += 140
		}
		if strings.HasSuffix(v, ".pkg") {
			score += 120
		}
		if strings.HasSuffix(v, ".zip") || strings.HasSuffix(v, ".tar.gz") || strings.HasSuffix(v, ".tgz") {
			score += 30
		}
		if strings.Contains(v, "universal") {
			score += 18
		}
	default:
		return 0
	}
	arch := sys.Arch
	if arch == "" {
		arch = runtime.GOARCH
	}
	if strings.Contains(v, "amd64") || strings.Contains(v, "x86_64") || strings.Contains(v, "x64") {
		if arch == "amd64" {
			score += 25
		}
	}
	if strings.Contains(v, "arm64") || strings.Contains(v, "aarch64") {
		if arch == "arm64" {
			score += 25
		}
	}
	return score
}

func wrongArchForSystem(v, arch string) bool {
	if arch == "" {
		arch = runtime.GOARCH
	}
	switch arch {
	case "amd64", "x86_64", "x64":
		return strings.Contains(v, "arm64") || strings.Contains(v, "aarch64") || strings.Contains(v, "armv7") || strings.Contains(v, "i686") || strings.Contains(v, "386")
	case "arm64", "aarch64":
		return strings.Contains(v, "x86_64") || strings.Contains(v, "amd64") || strings.Contains(v, "x64") || strings.Contains(v, "i686") || strings.Contains(v, "386")
	case "386", "i386", "i686":
		return strings.Contains(v, "arm64") || strings.Contains(v, "aarch64") || strings.Contains(v, "x86_64") || strings.Contains(v, "amd64") || strings.Contains(v, "x64")
	}
	return false
}

const maxDownloadSize int64 = 4 << 30

func downloadFile(rawurl, path string) error {
	var last error
	for attempt := 1; attempt <= 3; attempt++ {
		if err := downloadFileOnce(rawurl, path); err != nil {
			last = err
			time.Sleep(time.Duration(attempt) * 700 * time.Millisecond)
			continue
		}
		return nil
	}
	return last
}

func downloadTimeout() time.Duration {
	if v := strings.TrimSpace(os.Getenv("INSTALLY_DOWNLOAD_TIMEOUT_SECONDS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return 10 * time.Minute
}

type progressReader struct {
	reader   io.Reader
	total    int64
	done     int64
	interval time.Time
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.done += int64(n)
	if IsVerbose() && time.Since(pr.interval) > 500*time.Millisecond {
		pr.interval = time.Now()
		pct := float64(pr.done) / float64(pr.total) * 100
		if pr.total > 0 {
			fmt.Fprintf(os.Stderr, "\r  ↓ %s / %s (%.0f%%)", humanSize(pr.done), humanSize(pr.total), pct)
		} else {
			fmt.Fprintf(os.Stderr, "\r  ↓ %s", humanSize(pr.done))
		}
	}
	return n, err
}

func downloadFileOnce(rawurl, path string) error {
	if _, err := validateDownloadURL(rawurl); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", rawurl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Instally-Go")
	req.Header.Set("Accept", "application/octet-stream,*/*")
	client := &http.Client{
		Timeout: downloadTimeout(),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 8 {
				return fmt.Errorf("too many redirects")
			}
			_, err := validateDownloadURL(req.URL.String())
			return err
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download returned %s", resp.Status)
	}
	if resp.ContentLength > maxDownloadSize {
		return fmt.Errorf("download is too large: %s", humanSize(resp.ContentLength))
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".part"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	hasher := sha256.New()
	limited := io.LimitReader(resp.Body, maxDownloadSize+1)
	var reader io.Reader = limited
	if IsVerbose() && resp.ContentLength > 0 {
		reader = &progressReader{reader: limited, total: resp.ContentLength}
	}
	multi := io.MultiWriter(f, hasher)
	n, copyErr := io.Copy(multi, reader)
	if IsVerbose() && resp.ContentLength > 0 {
		fmt.Fprintf(os.Stderr, "\r  ↓ %s / %s (100%%)\n", humanSize(n), humanSize(resp.ContentLength))
	}
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	if n > maxDownloadSize {
		_ = os.Remove(tmp)
		return fmt.Errorf("download exceeded max size: %s", humanSize(maxDownloadSize))
	}
	if resp.ContentLength >= 0 && n != resp.ContentLength {
		_ = os.Remove(tmp)
		return fmt.Errorf("download size mismatch: got %s, expected %s", humanSize(n), humanSize(resp.ContentLength))
	}
	downloadedSHA := hex.EncodeToString(hasher.Sum(nil))
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	if err := writeDownloadIntegrity(path, downloadedSHA); err != nil {
		return err
	}
	return nil
}

func writeDownloadIntegrity(path, sha string) error {
	sumPath := path + ".sha256"
	return os.WriteFile(sumPath, []byte(sha+"  "+filepath.Base(path)+"\n"), 0o600)
}
