package app

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"
)

func validateInstallItems(kind string, items []string) ([]string, []string) {
	valid := make([]string, 0, len(items))
	var warnings []string
	for _, raw := range items {
		item := strings.TrimSpace(raw)
		if item == "" {
			continue
		}
		if err := validateInstallItem(kind, item); err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %q отклонён: %v", kind, compact(item, item), err))
			continue
		}
		valid = append(valid, item)
	}
	return valid, warnings
}

func validateInstallItem(kind, item string) error {
	if len(item) > 512 {
		return fmt.Errorf("слишком длинное значение")
	}
	if strings.HasPrefix(item, "-") {
		return fmt.Errorf("значение начинается с '-' и может быть воспринято как опция")
	}
	if strings.ContainsAny(item, "\x00\n\r\t") {
		return fmt.Errorf("обнаружены управляющие символы")
	}
	for _, r := range item {
		if unicode.IsControl(r) {
			return fmt.Errorf("обнаружен управляющий символ U+%04X", r)
		}
		if isBidiControl(r) {
			return fmt.Errorf("обнаружен bidi/control символ U+%04X", r)
		}
	}
	if strings.ContainsAny(item, ";|`$<>") {
		return fmt.Errorf("обнаружены shell-подобные спецсимволы")
	}
	switch kind {
	case "pkg", "aur", "flatpak", "snap", "pip", "pipx", "npm", "cargo", "go", "winget", "scoop", "choco", "brew", "mas":
		return validatePackageLikeName(item)
	}
	return nil
}

func validatePackageLikeName(item string) error {
	if strings.ContainsAny(item, " ") {
		return fmt.Errorf("имя пакета/приложения не должно содержать пробелы; используй кавычки только для путей к файлам")
	}
	for _, r := range item {
		ok := unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("._+@:/#=-", r)
		if !ok {
			return fmt.Errorf("недопустимый символ в имени: %q", r)
		}
	}
	return nil
}

func isBidiControl(r rune) bool {
	switch r {
	case '\u202a', '\u202b', '\u202c', '\u202d', '\u202e', '\u2066', '\u2067', '\u2068', '\u2069':
		return true
	}
	return false
}

func validateGitTarget(raw string) error {
	item := strings.TrimSpace(raw)
	if ownerRepoRE.MatchString(item) {
		return nil
	}
	if strings.ContainsAny(item, "\x00\n\r\t;|`$<>") {
		return fmt.Errorf("git target contains unsafe characters")
	}
	if strings.HasPrefix(item, "git@") && strings.Contains(item, ":") {
		return nil
	}
	u, err := url.Parse(item)
	if err != nil {
		return err
	}
	if u.User != nil {
		return fmt.Errorf("credentials in git URLs are not allowed")
	}
	switch u.Scheme {
	case "https", "ssh":
		if u.Hostname() == "" {
			return fmt.Errorf("git URL host is empty")
		}
		return nil
	case "http", "git":
		if boolEnv("INSTALLY_ALLOW_INSECURE_GIT") {
			if u.Hostname() == "" {
				return fmt.Errorf("git URL host is empty")
			}
			return nil
		}
		return fmt.Errorf("insecure git scheme %q is blocked by default; use https/ssh or set INSTALLY_ALLOW_INSECURE_GIT=1 for a trusted mirror", u.Scheme)
	}
	return fmt.Errorf("unsupported git URL scheme: %s", u.Scheme)
}
