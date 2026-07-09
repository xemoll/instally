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
	return 0
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
	if len(releases[0].Assets) == 0 {
		info.Error = "latest release has no assets"
		return info
	}
	best := releases[0].Assets[0]
	info.AssetName = best.Name
	info.AssetURL = best.BrowserDownloadURL
	info.Size = best.Size
	return info
}

func SelfPlan(info UpdateInfo, newBinary string) Plan {
	cmds := []CommandSpec{
		{
			Title: fmt.Sprintf("Download Instally %s (%s)", info.Latest, info.AssetName),
			Shell: fmt.Sprintf("curl -fL -o %s %s", shellQuote(newBinary), shellQuote(info.AssetURL)),
		},
		{
			Title: "Make binary executable",
			Shell: fmt.Sprintf("chmod +x %s", shellQuote(newBinary)),
		},
	}
	self := SelfPath()
	dst := self
	isInHome := strings.HasPrefix(self, homeDir())

	if !isInHome && self != "" {
		cmds = append(cmds, CommandSpec{
			Title: fmt.Sprintf("Install new binary to %s", self),
			Shell: fmt.Sprintf("cp %s %s && chmod +x %s && rm %s",
				shellQuote(newBinary), shellQuote(self), shellQuote(self), shellQuote(newBinary)),
			Admin: true,
		})
	} else {
		if self == "" {
			dst = filepath.Join(homeDir(), ".local", "bin", "instally")
		}
		cmds = append(cmds, CommandSpec{
			Title: fmt.Sprintf("Replace binary at %s", dst),
			Shell: fmt.Sprintf("cp %s %s && chmod +x %s && touch -r %s %s 2>/dev/null; rm %s",
				shellQuote(newBinary), shellQuote(dst), shellQuote(dst),
				shellQuote(dst), shellQuote(self), shellQuote(newBinary)),
		})
	}

	return Plan{
		System:   Detect(),
		Commands: cmds,
		Warnings: []string{},
	}
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
	_ = os.MkdirAll(tmpDir, 0o700)

	dl := filepath.Join(tmpDir, info.AssetName)
	integrityPath := dl + ".sha256"
	_ = os.Remove(dl)
	_ = os.Remove(integrityPath)

	if opts.DryRun {
		fmt.Fprintf(&out, "would download: %s\n", info.AssetURL)
		fmt.Fprintf(&out, "would save to: %s\n", dl)
		fmt.Fprintf(&out, "would replace: %s\n\n", SelfPath())
		fmt.Fprintf(&out, "Instally: готово (dry-run)\n")
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
		_ = os.Remove(dl)
		res.OK = false
		res.Errors = append(res.Errors, "security scan error")
		res.Output = out.String()
		return res
	}
	if rep.Status == "unsafe" && !opts.AllowUnknown {
		fmt.Fprintf(&out, "update binary flagged as unsafe, use --allow-unknown to force\n")
		_ = os.Remove(dl)
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
	fmt.Fprintf(&out, "Backing up current binary to %s\n", filepath.Base(backup))
	_ = os.Rename(self, backup)

	fmt.Fprintf(&out, "Installing %s → %s\n", dl, self)
	if err := copyFile(dl, self); err != nil {
		fmt.Fprintf(&out, "install failed: %v\n", err)
		_ = os.Rename(backup, self)
		res.OK = false
		res.Errors = append(res.Errors, err.Error())
		res.Output = out.String()
		return res
	}
	_ = os.Chmod(self, 0o755)
	_ = os.Remove(dl)
	_ = os.Remove(integrityPath)
	_ = os.Remove(backup)

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
		_ = os.Remove(tmp)
		return err
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}

	return os.Rename(tmp, dst)
}
