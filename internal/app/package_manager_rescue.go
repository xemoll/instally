package app

import "strings"

// withManagerRefresh attaches a safe metadata/source refresh command to
// package-manager install commands. The runner uses it only after the first
// install attempt fails, then retries the original install once.
func withManagerRefresh(c CommandSpec, m Manager) CommandSpec {
	if len(c.Refresh) == 0 {
		c.Refresh = managerRefreshCommand(m.ID)
	}
	return c
}

func managerRefreshCommand(manager string) []string {
	switch strings.ToLower(manager) {
	case "apt":
		return []string{"apt-get", "update"}
	case "pacman":
		// Refresh package databases only on failure. Full -Syu is intentionally
		// not forced because Instally must not surprise-upgrade the whole system.
		return []string{"pacman", "-Sy"}
	case "dnf":
		return []string{"dnf", "makecache", "--refresh"}
	case "zypper":
		return []string{"zypper", "refresh"}
	case "apk":
		return []string{"apk", "update"}
	case "xbps":
		return []string{"xbps-install", "-S"}
	case "eopkg":
		return []string{"eopkg", "ur"}
	case "emerge":
		return []string{"emerge", "--sync"}
	case "packagekit":
		return []string{"pkcon", "refresh"}
	case "brew":
		return []string{"brew", "update"}
	case "port":
		return []string{"port", "selfupdate"}
	case "winget":
		return []string{"winget", "source", "update"}
	case "scoop":
		return []string{"scoop", "update"}
	case "choco":
		return []string{"choco", "source", "list"}
	default:
		return nil
	}
}

func refreshHint(manager string) string {
	switch strings.ToLower(manager) {
	case "apt":
		return "apt cache may be stale or repositories may be disabled; Instally refreshed apt metadata and retried once"
	case "pacman":
		return "pacman sync database may be stale; Instally refreshed package databases and retried once"
	case "dnf":
		return "dnf metadata may be stale; Instally refreshed metadata and retried once"
	case "zypper":
		return "zypper repositories may need refresh; Instally refreshed repositories and retried once"
	case "apk":
		return "apk indexes may be stale; Instally updated indexes and retried once"
	case "xbps":
		return "xbps repositories may be stale; Instally synchronized repositories and retried once"
	case "eopkg":
		return "eopkg repository metadata may be stale; Instally updated repositories and retried once"
	case "emerge":
		return "Portage tree may be stale; Instally synchronized it and retried once"
	case "packagekit":
		return "PackageKit metadata may be stale; Instally refreshed it and retried once"
	case "brew":
		return "Homebrew metadata may be stale; Instally ran brew update and retried once"
	case "port":
		return "MacPorts metadata may be stale; Instally ran selfupdate and retried once"
	case "winget":
		return "WinGet sources may be stale; Instally updated sources and retried once"
	case "scoop":
		return "Scoop buckets may be stale; Instally updated them and retried once"
	case "choco":
		return "Chocolatey source metadata was checked before retry; verify enabled sources if the package is still missing"
	default:
		return "package-manager metadata was refreshed and the command was retried once"
	}
}

func managerFromCommand(c CommandSpec) string {
	if len(c.Cmd) > 0 {
		prog := strings.ToLower(c.Cmd[0])
		switch prog {
		case "apt", "apt-get":
			return "apt"
		case "pacman":
			return "pacman"
		case "dnf", "dnf5":
			return "dnf"
		case "zypper":
			return "zypper"
		case "apk":
			return "apk"
		case "xbps-install":
			return "xbps"
		case "eopkg":
			return "eopkg"
		case "emerge":
			return "emerge"
		case "pkcon":
			return "packagekit"
		case "brew":
			return "brew"
		case "port":
			return "port"
		case "winget":
			return "winget"
		case "scoop":
			return "scoop"
		case "choco":
			return "choco"
		}
	}
	return ""
}

func refreshLine(c CommandSpec) string {
	if len(c.Refresh) == 0 {
		return ""
	}
	return "on failure: " + shellJoin(c.Refresh) + " && retry once"
}

func diagnoseCommandFailure(c CommandSpec, text string) string {
	lower := strings.ToLower(text)
	manager := managerFromCommand(c)
	parts := []string{}
	if manager != "" && len(c.Refresh) > 0 {
		parts = append(parts, refreshHint(manager))
	}
	if strings.Contains(lower, "unable to locate package") || strings.Contains(lower, "no package") || strings.Contains(lower, "not found") || strings.Contains(lower, "no match") {
		parts = append(parts, "package still was not found after retry; check the exact package/app id or enable the required repository/source")
	}
	if strings.Contains(lower, "timed out") || strings.Contains(lower, "deadline exceeded") || strings.Contains(lower, "connection timed out") {
		parts = append(parts, "package-manager refresh or install timed out; check internet access, VPN/proxy, DNS and mirrors before retrying")
	}
	if strings.Contains(lower, "could not resolve") || strings.Contains(lower, "temporary failure") || strings.Contains(lower, "name or service not known") || strings.Contains(lower, "dns") {
		parts = append(parts, "network/DNS problem detected; check VPN, proxy, DNS and package-manager mirrors")
	}
	if strings.Contains(lower, "permission denied") || strings.Contains(lower, "are you root") || strings.Contains(lower, "need root") || strings.Contains(lower, "requires root") {
		parts = append(parts, "permission problem detected; run from a terminal so sudo can ask for the password, or use a desktop session with pkexec")
	}
	if strings.Contains(lower, "could not get lock") || strings.Contains(lower, "unable to acquire") || strings.Contains(lower, "database is locked") || strings.Contains(lower, "lock file") {
		parts = append(parts, "package-manager lock detected; close other installers/updaters and retry")
	}
	if strings.Contains(lower, "404") || strings.Contains(lower, "mirror") || strings.Contains(lower, "checksum") || strings.Contains(lower, "signature") || strings.Contains(lower, "gpg") {
		parts = append(parts, "mirror/signature problem detected; refresh sources, check time/date, and verify trusted repositories")
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(appendUnique(nil, parts...), "; ")
}
