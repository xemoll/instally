package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"
)

var configMu sync.Mutex

type UserConfig struct {
	VirusTotalAPIKey string `json:"virus_total_api_key,omitempty"`
	Language         string `json:"language,omitempty"`
}

func configPath() string { return filepath.Join(dataDir(), "config.json") }

func LoadConfig() UserConfig {
	b, err := readFileWithTimeout(configPath(), 5*time.Second)
	if err != nil {
		return UserConfig{}
	}
	var c UserConfig
	_ = json.Unmarshal(b, &c)
	c.Language = normalizeLang(c.Language)
	return c
}

func SaveConfig(c UserConfig) error {
	configMu.Lock()
	defer configMu.Unlock()
	if err := os.MkdirAll(dataDir(), 0o700); err != nil {
		return err
	}
	c.Language = normalizeLang(c.Language)
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := configPath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, configPath())
}

func SaveVirusTotalKey(key string) error {
	key = strings.TrimSpace(key)
	if err := validateVirusTotalKey(key); err != nil {
		return err
	}
	c := LoadConfig()
	c.VirusTotalAPIKey = key
	return SaveConfig(c)
}

func validateVirusTotalKey(key string) error {
	if key == "" {
		return fmt.Errorf("VirusTotal API key is empty")
	}
	for _, r := range key {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return fmt.Errorf("VirusTotal API key contains whitespace/control characters")
		}
	}
	return nil
}

func ClearVirusTotalKey() error {
	c := LoadConfig()
	c.VirusTotalAPIKey = ""
	return SaveConfig(c)
}

func VirusTotalStatus() string {
	cfg := LoadConfig()
	key := strings.TrimSpace(os.Getenv("INSTALLY_VT_API_KEY"))
	source := "not configured"
	configured := false
	if key != "" {
		configured = true
		source = "environment INSTALLY_VT_API_KEY"
	} else if path := strings.TrimSpace(os.Getenv("INSTALLY_VT_KEY_FILE")); path != "" && fileHasText(path) {
		configured = true
		source = "secure key file"
	} else if strings.TrimSpace(cfg.VirusTotalAPIKey) != "" {
		configured = true
		source = "saved user config"
	}
	upload := boolEnv("INSTALLY_VT_UPLOAD")
	large := vtMaxUploadSize()
	if AppLanguage() == "en" {
		if !configured {
			return fmt.Sprintf("VirusTotal: not configured\nSet INSTALLY_VT_API_KEY or run: instally --vt-save-key <key>\nUploads are opt-in with --vt-upload or INSTALLY_VT_UPLOAD=1. Large upload limit: %s\n", humanSize(large))
		}
		return fmt.Sprintf("VirusTotal: configured (%s)\nUpload enabled: %v\nLarge upload limit: %s\n", source, upload, humanSize(large))
	}
	if !configured {
		return fmt.Sprintf("VirusTotal: не настроен\nУкажи INSTALLY_VT_API_KEY или выполни: instally --vt-save-key <key>\nОтправка файлов включается только через --vt-upload или INSTALLY_VT_UPLOAD=1. Лимит upload: %s\n", humanSize(large))
	}
	return fmt.Sprintf("VirusTotal: настроен (%s)\nUpload включён: %v\nЛимит upload: %s\n", source, upload, humanSize(large))
}

func VirusTotalSelfTestWithConfiguredKey() string {
	key := strings.TrimSpace(os.Getenv("INSTALLY_VT_API_KEY"))
	if key == "" {
		key = readKeyFile(strings.TrimSpace(os.Getenv("INSTALLY_VT_KEY_FILE")))
	}
	if key == "" {
		key = strings.TrimSpace(LoadConfig().VirusTotalAPIKey)
	}
	if key == "" {
		if AppLanguage() == "en" {
			return "VirusTotal test: no API key configured\n"
		}
		return "VirusTotal test: API-ключ не настроен\n"
	}
	stats, found, detail := vtLookupHash(eicarSHA256, key)
	if found {
		c := vtStatsCheck(stats, "EICAR hash lookup")
		return fmt.Sprintf("VirusTotal test: OK (%s)\n", c.Detail)
	}
	return "VirusTotal test: " + detail + "\n"
}

func normalizeLang(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "_", "-")
	if strings.HasPrefix(s, "en") {
		return "en"
	}
	if strings.HasPrefix(s, "ru") || s == "" {
		return "ru"
	}
	return "ru"
}

func AppLanguage() string {
	if raw := strings.TrimSpace(os.Getenv("INSTALLY_LANG")); raw != "" {
		return normalizeLang(raw)
	}
	return normalizeLang(LoadConfig().Language)
}

func SetAppLanguage(lang string) { _ = os.Setenv("INSTALLY_LANG", normalizeLang(lang)) }

func SaveLanguage(lang string) error {
	c := LoadConfig()
	c.Language = normalizeLang(lang)
	return SaveConfig(c)
}

func readKeyFile(path string) string {
	if path == "" {
		return ""
	}
	b, err := readFileWithTimeout(path, 5*time.Second)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func fileHasText(path string) bool { return readKeyFile(path) != "" }

func IsEnglish() bool { return AppLanguage() == "en" }

func T(key string) string {
	lang := AppLanguage()
	ru := map[string]string{
		"app.name":           "Instally",
		"source.placeholder": "https://example.com/app.AppImage",
		"source.title":       "Ссылка для установки",
		"source.hint":        "Ссылка, GitHub, имя программы или локальный файл",
		"choose.file":        "Выбрать файл",
		"install.safe":       "Проверить и установить",
		"ready":              "Готово",
		"advanced":           "Дополнительно",
		"log":                "Журнал",
		"vt.key.placeholder": "VirusTotal API key — необязательно",
		"vt.upload":          "разрешить отправку файла в VirusTotal",
		"allow.limited":      "разрешить неполную проверку",
		"show.plan":          "Показать план",
		"scan.only":          "Только проверить",
		"language":           "Язык",
		"plan.placeholder":   "План появится здесь после распознавания источника.",
		"log.placeholder":    "Журнал появится во время проверки или установки",
		"add.source":         "Добавь файл, ссылку, GitHub или имя программы.",
		"checking":           "Проверяем источник…",
		"installing.checked": "Устанавливаем проверенное…",
	}
	en := map[string]string{
		"app.name":           "Instally",
		"source.placeholder": "https://example.com/app.AppImage",
		"source.title":       "Install link",
		"source.hint":        "Link, GitHub repo, app name, or local file",
		"choose.file":        "Choose file",
		"install.safe":       "Check and install",
		"ready":              "Ready",
		"advanced":           "Advanced",
		"log":                "Log",
		"vt.key.placeholder": "VirusTotal API key — optional",
		"vt.upload":          "allow uploading files to VirusTotal",
		"allow.limited":      "allow limited checks",
		"show.plan":          "Show plan",
		"scan.only":          "Scan only",
		"language":           "Language",
		"plan.placeholder":   "The plan will appear after source detection.",
		"log.placeholder":    "Log output appears during scan or install",
		"add.source":         "Add a file, URL, GitHub repo, or app name.",
		"checking":           "Checking source…",
		"installing.checked": "Installing checked source…",
	}
	if lang == "en" {
		if v := en[key]; v != "" {
			return v
		}
	}
	if v := ru[key]; v != "" {
		return v
	}
	return key
}
