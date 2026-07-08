package app

import (
	"bufio"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var ownerRepoRE = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

func ParseBatchText(text string) []Task {
	var tasks []Task
	s := bufio.NewScanner(strings.NewReader(text))
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			kind := normalizeKind(parts[0])
			items := normalizeExplicitItems(kind, fields(parts[1]))
			if kind != "" && len(items) > 0 {
				tasks = append(tasks, Task{Kind: kind, Items: items})
				continue
			}
		}
		for _, item := range fields(line) {
			tasks = append(tasks, AutoTask(item))
		}
	}
	return mergeTasks(tasks)
}

func normalizeExplicitItems(kind string, items []string) []string {
	if len(items) == 0 {
		return items
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		switch kind {
		case "github", "release":
			out = append(out, normalizeGitHubTarget(item))
		case "local":
			out = append(out, expandPath(item))
		default:
			out = append(out, item)
		}
	}
	return out
}

func ParseMultiItems(items ...string) []Task {
	var b strings.Builder
	for _, item := range items {
		for _, part := range splitMultiItem(item) {
			part = strings.TrimSpace(part)
			if part != "" {
				b.WriteString(part)
				b.WriteByte('\n')
			}
		}
	}
	return ParseBatchText(b.String())
}

func splitMultiItem(s string) []string {
	var out []string
	var b strings.Builder
	quote := rune(0)
	for _, r := range s {
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				b.WriteRune(r)
			}
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			continue
		}
		if r == ',' || r == '\n' || r == '\r' {
			v := strings.TrimSpace(b.String())
			if v != "" {
				out = append(out, v)
			}
			b.Reset()
			continue
		}
		b.WriteRune(r)
	}
	if v := strings.TrimSpace(b.String()); v != "" {
		out = append(out, v)
	}
	return out
}

func ParseBatchFile(path string) ([]Task, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseBatchText(string(b)), nil
}

func AutoTask(raw string) Task {
	item := strings.TrimSpace(raw)
	if item == "" {
		return Task{}
	}
	lower := strings.ToLower(item)
	if isWindowsPath(item) || fileExists(item) || strings.HasPrefix(item, ".") || strings.HasPrefix(item, "~") || filepath.IsAbs(item) || looksLocalInstallerName(item) {
		return Task{Kind: "local", Items: []string{expandPath(item)}}
	}
	if key, ok := knownAppKey(lower); ok {
		return Task{Kind: "app", Items: []string{key}}
	}
	if strings.HasPrefix(lower, "gh:") {
		return Task{Kind: "github", Items: []string{normalizeGitHubTarget(item)}}
	}
	if strings.HasPrefix(lower, "github.com/") || strings.HasPrefix(lower, "www.github.com/") {
		return Task{Kind: "github", Items: []string{normalizeGitHubTarget(item)}}
	}
	if strings.Contains(item, ":") && !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		parts := strings.SplitN(item, ":", 2)
		kind := normalizeKind(parts[0])
		if kind != "" {
			return Task{Kind: kind, Items: []string{strings.TrimSpace(parts[1])}}
		}
	}
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		if isGitURL(item) && !looksDownload(item) {
			if isGitHubRepoURL(item) {
				return Task{Kind: "github", Items: []string{normalizeGitHubTarget(item)}}
			}
			return Task{Kind: "git", Items: []string{normalizeGitURL(item)}}
		}
		return Task{Kind: "url", Items: []string{item}}
	}
	if ownerRepoRE.MatchString(item) {
		return Task{Kind: "github", Items: []string{item}}
	}
	return Task{Kind: "pkg", Items: []string{item}}
}

func normalizeKind(k string) string {
	k = strings.ToLower(strings.TrimSpace(k))
	switch k {
	case "pkg", "native", "repo", "package", "packages":
		return "pkg"
	case "aur":
		return "aur"
	case "flatpak", "flathub":
		return "flatpak"
	case "snap":
		return "snap"
	case "pip", "pipx", "npm", "cargo", "go", "brew", "mas", "winget", "scoop", "choco", "app", "preset", "profile", "ai-tools", "official-ollama", "official-opencode", "official-claude-code", "official-firefox", "official-discord":
		if k == "profile" {
			return "preset"
		}
		return k
	case "github", "gh":
		return "github"
	case "git", "gitlab", "codeberg", "clone":
		return "git"
	case "release", "github-release", "gh-release":
		return "release"
	case "local", "file", "appimage", "deb", "rpm", "apk", "msi", "exe", "dmg", "pkgfile", "url":
		if k == "file" || k == "appimage" || k == "deb" || k == "rpm" || k == "apk" || k == "msi" || k == "exe" || k == "dmg" || k == "pkgfile" {
			return "local"
		}
		if k == "profile" {
			return "preset"
		}
		return k
	case "auto":
		return "auto"
	}
	return ""
}

func fields(s string) []string {
	var out []string
	var b strings.Builder
	quote := rune(0)
	escaped := false
	flush := func() {
		v := strings.TrimSpace(b.String())
		if v != "" {
			out = append(out, v)
		}
		b.Reset()
	}
	for _, r := range s {
		if escaped {
			b.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				b.WriteRune(r)
			}
			continue
		}
		if r == '"' || r == '\'' {
			quote = r
			continue
		}
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			flush()
			continue
		}
		b.WriteRune(r)
	}
	flush()
	return out
}

func mergeTasks(tasks []Task) []Task {
	m := map[string][]string{}
	order := []string{}
	for _, t := range tasks {
		if t.Kind == "" || len(t.Items) == 0 {
			continue
		}
		if t.Kind == "auto" {
			for _, item := range t.Items {
				a := AutoTask(item)
				if a.Kind != "" {
					if _, ok := m[a.Kind]; !ok {
						order = append(order, a.Kind)
					}
					m[a.Kind] = appendUnique(m[a.Kind], a.Items...)
				}
			}
			continue
		}
		if _, ok := m[t.Kind]; !ok {
			order = append(order, t.Kind)
		}
		m[t.Kind] = appendUnique(m[t.Kind], t.Items...)
	}
	out := make([]Task, 0, len(order))
	for _, k := range order {
		out = append(out, Task{Kind: k, Items: m[k]})
	}
	return out
}

func fileExists(p string) bool {
	_, err := os.Stat(expandPath(p))
	return err == nil
}

func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") || p == "~" {
		return filepath.Join(homeDir(), strings.TrimPrefix(p, "~/"))
	}
	return p
}

func isGitURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	h := strings.ToLower(u.Host)
	return strings.Contains(h, "github.com") || strings.Contains(h, "gitlab.com") || strings.Contains(h, "codeberg.org") || strings.Contains(h, "bitbucket.org") || strings.Contains(h, "sr.ht")
}

func looksDownload(raw string) bool {
	path := strings.ToLower(raw)
	for _, ext := range []string{".deb", ".rpm", ".apk", ".msi", ".exe", ".appx", ".msix", ".pkg", ".dmg", ".appimage", ".zip", ".7z", ".tar.gz", ".tgz", ".tar.xz", ".tar.bz2", ".tar.zst", ".pkg.tar.zst", ".pkg.tar.xz", ".flatpakref", ".flatpakrepo", ".run", ".bin"} {
		if strings.Contains(path, ext) {
			return true
		}
	}
	return false
}

func looksLocalInstallerName(raw string) bool {
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return false
	}
	return looksDownload(lower) || strings.HasSuffix(lower, ".run") || strings.HasSuffix(lower, ".sh")
}

func isWindowsPath(raw string) bool {
	if len(raw) < 3 {
		return false
	}
	c := raw[0]
	return ((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) && raw[1] == ':' && (raw[2] == '\\' || raw[2] == '/')
}

func normalizeGitURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) >= 2 {
		return "https://" + u.Host + "/" + parts[0] + "/" + strings.TrimSuffix(parts[1], ".git") + ".git"
	}
	return raw
}

func isGitHubRepoURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if strings.ToLower(u.Host) != "github.com" {
		return false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	return len(parts) >= 2 && parts[0] != "" && parts[1] != ""
}

func githubOwnerRepoFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) >= 2 {
		return parts[0] + "/" + strings.TrimSuffix(parts[1], ".git")
	}
	return raw
}

func normalizeGitHubTarget(raw string) string {
	item := strings.TrimSpace(raw)
	lower := strings.ToLower(item)
	if strings.HasPrefix(lower, "gh:") {
		item = strings.TrimSpace(item[3:])
		lower = strings.ToLower(item)
	}
	if strings.HasPrefix(lower, "github:") {
		item = strings.TrimSpace(item[len("github:"):])
		lower = strings.ToLower(item)
	}
	if ownerRepoRE.MatchString(item) {
		return item
	}
	if strings.HasPrefix(lower, "github.com/") || strings.HasPrefix(lower, "www.github.com/") {
		return githubOwnerRepoFromURL("https://" + item)
	}
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		if isGitHubRepoURL(item) {
			return githubOwnerRepoFromURL(item)
		}
	}
	return item
}
