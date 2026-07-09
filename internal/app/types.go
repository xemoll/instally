package app

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

type Family string

const (
	Linux   Family = "linux"
	Windows Family = "windows"
	Darwin  Family = "darwin"
	Unknown Family = "unknown"
)

type Manager struct {
	ID        string              `json:"id"`
	Label     string              `json:"label"`
	Family    Family              `json:"family"`
	Tools     []string            `json:"tools"`
	Install   []string            `json:"install"`
	Yes       []string            `json:"yes"`
	Update    []string            `json:"update"`
	Remove    []string            `json:"remove"`
	Search    []string            `json:"search"`
	Info      []string            `json:"info"`
	Local     map[string][]string `json:"local"`
	Prepare   []string            `json:"prepare"`
	Priority  int                 `json:"priority"`
	NeedsElev bool                `json:"needs_elevation"`
}

type SystemInfo struct {
	Family       Family            `json:"family"`
	GOOS         string            `json:"goos"`
	Arch         string            `json:"arch"`
	OSID         string            `json:"os_id"`
	OSLike       string            `json:"os_like"`
	Manager      Manager           `json:"manager"`
	ManagerFound bool              `json:"manager_found"`
	ToolPath     string            `json:"tool_path"`
	Tools        map[string]string `json:"tools"`
	Home         string            `json:"home"`
	BuildDir     string            `json:"build_dir"`
	DataDir      string            `json:"data_dir"`
	CacheDir     string            `json:"cache_dir"`
}

type Task struct {
	Kind  string   `json:"kind"`
	Items []string `json:"items"`
}

type CommandSpec struct {
	Title          string            `json:"title"`
	Cmd            []string          `json:"cmd,omitempty"`
	Shell          string            `json:"shell,omitempty"`
	Dir            string            `json:"dir,omitempty"`
	Admin          bool              `json:"admin,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	Refresh        []string          `json:"refresh_on_failure,omitempty"`
	TimeoutSeconds int               `json:"timeout_seconds,omitempty"`
}

type Plan struct {
	System          SystemInfo    `json:"system"`
	Tasks           []Task        `json:"tasks"`
	Commands        []CommandSpec `json:"commands"`
	Warnings        []string      `json:"warnings"`
	ContinueOnError bool          `json:"continue_on_error"`
}

type RunResult struct {
	DryRun   bool     `json:"dry_run"`
	OK       bool     `json:"ok"`
	ExitCode int      `json:"exit_code"`
	Output   string   `json:"output"`
	Plan     Plan     `json:"plan"`
	Errors   []string `json:"errors"`
}

func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil || h == "" {
		return "."
	}
	return h
}

func cacheDir() string {
	if v := os.Getenv("INSTALLY_CACHE_DIR"); v != "" {
		return v
	}
	if runtime.GOOS == "windows" {
		if v := os.Getenv("LOCALAPPDATA"); v != "" {
			return filepath.Join(v, "Instally", "Cache")
		}
	}
	if v := os.Getenv("XDG_CACHE_HOME"); v != "" {
		return filepath.Join(v, "instally")
	}
	return filepath.Join(homeDir(), ".cache", "instally")
}

func dataDir() string {
	if v := os.Getenv("INSTALLY_DATA_DIR"); v != "" {
		return v
	}
	if runtime.GOOS == "windows" {
		if v := os.Getenv("LOCALAPPDATA"); v != "" {
			return filepath.Join(v, "Instally")
		}
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(homeDir(), "Library", "Application Support", "Instally")
	}
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return filepath.Join(v, "instally")
	}
	return filepath.Join(homeDir(), ".local", "share", "instally")
}

func buildDir() string {
	if v := os.Getenv("INSTALLY_BUILD_DIR"); v != "" {
		return v
	}
	return filepath.Join(homeDir(), "Builds", "instally")
}

func localBin() string {
	if runtime.GOOS == "windows" {
		return dataDir()
	}
	return filepath.Join(homeDir(), ".local", "bin")
}

func commandExists(name string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	return ""
}

func commandLine(c CommandSpec) string {
	prefix := ""
	if len(c.Env) > 0 {
		keys := make([]string, 0, len(c.Env))
		for k := range c.Env {
			kl := strings.ToLower(k)
		if strings.Contains(kl, "key") || strings.Contains(kl, "token") || strings.Contains(kl, "secret") || strings.Contains(kl, "password") || strings.Contains(kl, "credential") || strings.Contains(kl, "auth") || strings.Contains(kl, "apikey") {
				keys = append(keys, k+"=***")
			} else {
				keys = append(keys, k+"="+shellQuote(c.Env[k]))
			}
		}
		sort.Strings(keys)
		prefix = strings.Join(keys, " ") + " "
	}
	if c.Shell != "" {
		return prefix + c.Shell
	}
	return prefix + shellJoin(c.Cmd)
}

func shellJoin(parts []string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, shellQuote(p))
	}
	return strings.Join(out, " ")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if strings.IndexFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("@%_+=:,./-", r))
	}) == -1 {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func appendUnique(dst []string, src ...string) []string {
	seen := map[string]bool{}
	for _, v := range dst {
		seen[v] = true
	}
	for _, v := range src {
		if v != "" && !seen[v] {
			dst = append(dst, v)
			seen[v] = true
		}
	}
	return dst
}

func JSON(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func normalizeExt(path string) string {
	p := strings.ToLower(path)
	for _, s := range []string{".pkg.tar.zst", ".pkg.tar.xz", ".pkg.tar.gz", ".tar.gz", ".tar.xz", ".tar.bz2", ".tar.zst", ".flatpakref", ".flatpakrepo", ".appimage"} {
		if strings.HasSuffix(p, s) {
			return s
		}
	}
	return strings.ToLower(filepath.Ext(path))
}

func ensureDirs() error {
	for _, p := range []string{cacheDir(), dataDir(), buildDir(), localBin(), filepath.Join(cacheDir(), "downloads"), filepath.Join(dataDir(), "appimages")} {
		perm := os.FileMode(0o755)
		if p == cacheDir() || strings.HasPrefix(p, cacheDir()) {
			perm = 0o700
		}
		if err := os.MkdirAll(p, perm); err != nil {
			return fmt.Errorf("create %s: %w", p, err)
		}
	}
	return nil
}
