package app

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type UpdateInfo struct {
	Available bool   `json:"available"`
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	AssetName string `json:"asset_name,omitempty"`
	AssetURL  string `json:"asset_url,omitempty"`
	Size      int64  `json:"size,omitempty"`
	Error     string `json:"error,omitempty"`
}

func CompareVersions(a, b string) int {
	va := strings.TrimPrefix(a, "v")
	vb := strings.TrimPrefix(b, "v")
	// Strip build metadata (+...) — not meaningful for comparison
	if idx := strings.IndexByte(va, '+'); idx >= 0 {
		va = va[:idx]
	}
	if idx := strings.IndexByte(vb, '+'); idx >= 0 {
		vb = vb[:idx]
	}
	// Split off pre-release suffix for separate comparison
	var preA, preB string
	if idx := strings.IndexByte(va, '-'); idx >= 0 {
		va, preA = va[:idx], va[idx+1:]
	}
	if idx := strings.IndexByte(vb, '-'); idx >= 0 {
		vb, preB = vb[:idx], vb[idx+1:]
	}
	pa := strings.Split(va, ".")
	pb := strings.Split(vb, ".")
	max := len(pa)
	if len(pb) > max {
		max = len(pb)
	}
	for i := 0; i < max; i++ {
		var na, nb int
		if i < len(pa) {
			na, _ = strconv.Atoi(strings.TrimSpace(pa[i]))
		}
		if i < len(pb) {
			nb, _ = strconv.Atoi(strings.TrimSpace(pb[i]))
		}
		if na < nb {
			return -1
		}
		if na > nb {
			return 1
		}
	}
	// Same core version — compare pre-release
	if preA == preB {
		return 0
	}
	if preA == "" {
		return 1 // release > pre-release
	}
	if preB == "" {
		return -1
	}
	if preA < preB {
		return -1
	}
	return 1
}

func SelfUpdateCheck() UpdateInfo {
	info := UpdateInfo{Current: appVersion}
	releases, err := fetchGitHubReleases("xemoll/instally")
	if err != nil {
		info.Error = fmt.Sprintf("GitHub API error: %v", err)
		return info
	}
	if len(releases) == 0 {
		info.Error = "no releases found"
		return info
	}
	latest := strings.TrimPrefix(releases[0].TagName, "v")
	info.Latest = latest
	switch CompareVersions(latest, appVersion) {
	case 1:
		info.Available = true
	case -1, 0:
		info.Available = false
		return info
	}
	sys := Detect()
	for _, a := range releases[0].Assets {
		s := scoreAssetForSystem(a.Name, a.BrowserDownloadURL, sys)
		if s > 0 {
			info.AssetName = a.Name
			info.AssetURL = a.BrowserDownloadURL
			info.Size = a.Size
			return info
		}
	}
	info.Error = fmt.Sprintf("no compatible binary asset found for %s/%s in release %s",
		sys.Family, sys.Arch, releases[0].TagName)
	return info
}

func SelfUpdate(opts Options, info UpdateInfo) RunResult {
	var out bytes.Buffer
	res := RunResult{DryRun: opts.DryRun, OK: true}

	fmt.Fprintf(&out, "Instally update: %s → %s\n", info.Current, info.Latest)
	if info.Error != "" {
		fmt.Fprintf(&out, "error: %s\n", info.Error)
		res.OK = false
		res.Output = out.String()
		return res
	}
	if !info.Available {
		fmt.Fprintf(&out, "Already up to date (v%s)\n", info.Current)
		res.Output = out.String()
		return res
	}

	fmt.Fprintf(&out, "Downloading %s (%s)\n", info.AssetName, humanSize(info.Size))
	tmpDir := filepath.Join(cacheDir(), "self-update")
	if err := os.MkdirAll(tmpDir, 0o700); err != nil {
		fmt.Fprintf(&out, "failed to create temp dir: %v\n", err)
		res.OK = false
		res.Errors = append(res.Errors, err.Error())
		res.Output = out.String()
		return res
	}

	dl := filepath.Join(tmpDir, info.AssetName)
	integrityPath := dl + ".sha256"
	os.Remove(dl)
	os.Remove(integrityPath)

	if opts.DryRun {
		fmt.Fprintf(&out, "would download: %s\n", info.AssetURL)
		fmt.Fprintf(&out, "would save to: %s\n", dl)
		fmt.Fprintf(&out, "would replace: %s\n\n", SelfPath())
		fmt.Fprintf(&out, "Instally: ready (dry-run)\n")
		res.Output = out.String()
		return res
	}

	if err := downloadFile(info.AssetURL, dl); err != nil {
		fmt.Fprintf(&out, "download failed: %v\n", err)
		res.OK = false
		res.Errors = append(res.Errors, err.Error())
		res.Output = out.String()
		return res
	}

	rep := ScanFile(dl, SecurityOptions{VirusTotalKey: opts.VirusTotalKey, VirusTotalUpload: opts.VirusTotalUpload, AllowUnknown: true})
	writeSecurityHuman(&out, rep)
	if rep.Status == "error" {
		fmt.Fprintf(&out, "security scan error, blocking update\n")
		os.Remove(dl)
		res.OK = false
		res.Errors = append(res.Errors, "security scan error")
		res.Output = out.String()
		return res
	}
	if rep.Status == "unsafe" && !opts.AllowUnknown {
		fmt.Fprintf(&out, "update binary flagged as unsafe, use --allow-unknown to force\n")
		os.Remove(dl)
		res.OK = false
		res.Errors = append(res.Errors, "binary flagged as unsafe")
		res.Output = out.String()
		return res
	}

	self := SelfPath()
	if self == "" || self == "instally" {
		self = filepath.Join(homeDir(), ".local", "bin", "instally")
	}
	backup := self + ".bak." + strconv.FormatInt(time.Now().Unix(), 10)

	if err := os.Rename(self, backup); err != nil {
		fmt.Fprintf(&out, "warning: could not back up current binary: %v\n", err)
		backup = ""
	} else {
		fmt.Fprintf(&out, "Backed up to %s\n", filepath.Base(backup))
	}

	fmt.Fprintf(&out, "Installing %s\n", self)
	if err := copyFile(dl, self); err != nil {
		fmt.Fprintf(&out, "install failed: %v\n", err)
		if backup != "" {
			if rbErr := os.Rename(backup, self); rbErr != nil {
				fmt.Fprintf(&out, "error restoring backup: %v\n", rbErr)
			}
		}
		res.OK = false
		res.Errors = append(res.Errors, err.Error())
		res.Output = out.String()
		return res
	}
	if err := os.Chmod(self, 0o755); err != nil {
		fmt.Fprintf(&out, "warning: could not set executable bit: %v\n", err)
	}
	os.Remove(dl)
	os.Remove(integrityPath)
	if backup != "" {
		os.Remove(backup)
	}

	fmt.Fprintf(&out, "\nUpdated to v%s\n", info.Latest)
	if !opts.Yes {
		fmt.Fprintf(&out, "Restart Instally to use the new version\n")
	}
	res.Output = out.String()
	return res
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	tmp := dst + ".tmp." + strconv.FormatInt(time.Now().UnixNano(), 36)
	out, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}

	_, err = io.Copy(out, in)
	closeErr := out.Close()
	if err != nil {
		os.Remove(tmp)
		return err
	}
	if closeErr != nil {
		os.Remove(tmp)
		return closeErr
	}

	return os.Rename(tmp, dst)
}
