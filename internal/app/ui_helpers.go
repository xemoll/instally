package app

import (
	"fmt"
	"strings"
)

func SystemLabelForUI(sys SystemInfo) string {
	manager := sys.Manager.ID
	if manager == "" {
		manager = "none"
	}
	family := string(sys.Family)
	if family == "" || family == "unknown" {
		family = sys.GOOS
	}
	return family + " · " + manager
}

func SourceKindFriendly(kind string) string {
	switch kind {
	case "url":
		return "Ссылка на файл"
	case "github", "release":
		return "GitHub Release"
	case "git":
		return "Git-репозиторий"
	case "local":
		return "Локальный файл"
	case "app":
		return "Приложение"
	case "pkg":
		return "Пакет системы"
	case "flatpak":
		return "Flatpak"
	case "snap":
		return "Snap"
	case "aur":
		return "AUR"
	case "winget":
		return "Winget"
	case "scoop":
		return "Scoop"
	case "choco":
		return "Chocolatey"
	case "brew":
		return "Homebrew"
	case "pipx":
		return "pipx"
	case "npm":
		return "npm"
	case "cargo":
		return "Cargo"
	case "go":
		return "Go"
	default:
		if kind == "" {
			return "Источник"
		}
		return kind
	}
}

func PlanLinesForUI(plan Plan, max int) []string {
	if max <= 0 {
		max = len(plan.Commands)
	}
	lines := make([]string, 0, len(plan.Commands))
	for i, c := range plan.Commands {
		if i >= max {
			lines = append(lines, fmt.Sprintf("…и ещё %d шагов", len(plan.Commands)-i))
			break
		}
		cmd := strings.TrimSpace(commandLine(c))
		if cmd != "" {
			lines = append(lines, fmt.Sprintf("%d. %s\n   %s", i+1, c.Title, cmd))
		} else {
			lines = append(lines, fmt.Sprintf("%d. %s", i+1, c.Title))
		}
	}
	if len(lines) == 0 {
		lines = append(lines, "План пока пуст: источник не распознан или для него нет команды установки")
	}
	return lines
}

func PlanSummaryForUI(plan Plan) string {
	count := len(plan.Commands)
	warn := len(plan.Warnings)
	if count == 0 && warn == 0 {
		return "Команды установки пока не построены."
	}
	parts := []string{}
	if count == 1 {
		parts = append(parts, "1 шаг установки")
	} else if count > 1 {
		parts = append(parts, fmt.Sprintf("%d шагов установки", count))
	}
	if warn == 1 {
		parts = append(parts, "1 предупреждение")
	} else if warn > 1 {
		parts = append(parts, fmt.Sprintf("%d предупреждения", warn))
	}
	return strings.Join(parts, " · ")
}

func SecurityHumanTitleForUI(rep SecurityReport) string {
	switch rep.Status {
	case "clean":
		return "Всё выглядит нормально"
	case "unsafe":
		return "Устанавливать нельзя"
	case "error":
		return "Не удалось проверить"
	case "limited", "warning":
		return "Нужна ручная оценка"
	default:
		if rep.Title != "" {
			return rep.Title
		}
		return "Проверка завершена"
	}
}

func SecurityHumanSummaryForUI(rep SecurityReport) string {
	switch rep.Status {
	case "clean":
		return "Серьёзных угроз не найдено по доступным проверкам. Можно продолжить установку."
	case "unsafe":
		return "Найдены опасные признаки. Установка заблокирована."
	case "error":
		if rep.Summary != "" {
			return rep.Summary
		}
		return "Файл или источник не удалось проверить."
	case "limited", "warning":
		return "Проверка неполная. Устанавливай только если доверяешь источнику, либо включи разрешение неполной проверки."
	default:
		if rep.Summary != "" {
			return rep.Summary
		}
		return "Проверка завершена."
	}
}

func ShortSHAForUI(sha string) string {
	sha = strings.TrimSpace(sha)
	if sha == "" {
		return "—"
	}
	if len(sha) <= 30 {
		return sha
	}
	return sha[:16] + "…" + sha[len(sha)-10:]
}
