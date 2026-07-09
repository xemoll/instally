package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func InstallCommands(self string, full bool) []CommandSpec {
	if self == "" {
		self = "instally"
	}
	switch runtime.GOOS {
	case "windows":
		base := dataDir()
		dst := filepath.Join(base, "instally.exe")
		ps := fmt.Sprintf("New-Item -ItemType Directory -Force -Path %s | Out-Null; Copy-Item -Force %s %s", winPSQuote(base), winPSQuote(self), winPSQuote(dst))
		return []CommandSpec{{Title: "Install Instally for current Windows user", Shell: "powershell -NoProfile -ExecutionPolicy Bypass -Command " + winQuote(ps)}}
	case "darwin":
		bin := filepath.Join(homeDir(), ".local", "bin", "instally")
		app := filepath.Join(homeDir(), "Applications", "Instally.app")
		sh := fmt.Sprintf("mkdir -p %s %s/Contents/MacOS %s/Contents/Resources && cp %s %s && chmod +x %s && cat > %s/Contents/MacOS/Instally <<'EOS'\n#!/bin/sh\nexec %s --gui \"$@\"\nEOS\nchmod +x %s/Contents/MacOS/Instally && cat > %s/Contents/Info.plist <<'EOS'\n%s\nEOS", shellQuote(filepath.Dir(bin)), shellQuote(app), shellQuote(app), shellQuote(self), shellQuote(bin), shellQuote(bin), shellQuote(app), shellQuote(bin), shellQuote(app), shellQuote(app), macPlist())
		return []CommandSpec{{Title: "Install Instally.app", Shell: sh}}
	default:
		bin := filepath.Join(homeDir(), ".local", "bin", "instally")
		apps := filepath.Join(homeDir(), ".local", "share", "applications")
		desktop := filepath.Join(apps, "instally.desktop")
		mimeDir := filepath.Join(homeDir(), ".local", "share", "mime", "packages")
		mime := filepath.Join(mimeDir, "instally.xml")
		cmds := []CommandSpec{
			{Title: "Install Instally binary", Shell: fmt.Sprintf("mkdir -p %s %s && cp %s %s && chmod +x %s", shellQuote(filepath.Dir(bin)), shellQuote(apps), shellQuote(self), shellQuote(bin), shellQuote(bin))},
			{Title: "Install desktop entry", Shell: fmt.Sprintf("cat > %s <<'EOF'\n%s\nEOF", shellQuote(desktop), linuxDesktop(bin))},
			{Title: "Install MIME definitions", Shell: fmt.Sprintf("mkdir -p %s && cat > %s <<'EOF'\n%s\nEOF\nupdate-mime-database %s 2>/dev/null || true\nupdate-desktop-database %s 2>/dev/null || true", shellQuote(mimeDir), shellQuote(mime), mimeXML(), shellQuote(filepath.Join(homeDir(), ".local", "share", "mime")), shellQuote(apps))},
		}
		if full {
			cmds = append(cmds, SetDefaultCommands()...)
		}
		return cmds
	}
}

func SetDefaultCommands() []CommandSpec {
	switch runtime.GOOS {
	case "windows":
		bin := filepath.Join(dataDir(), "instally.exe")
		ps := fmt.Sprintf("New-Item -Path 'HKCU:\\Software\\Classes\\*\\shell\\Install with Instally\\command' -Force | Out-Null; Set-ItemProperty -Path 'HKCU:\\Software\\Classes\\*\\shell\\Install with Instally\\command' -Name '(default)' -Value %s", winPSQuote("\""+bin+"\" --open-file \"%1\""))
		return []CommandSpec{{Title: "Register Windows context menu", Shell: "powershell -NoProfile -ExecutionPolicy Bypass -Command " + winQuote(ps)}}
	case "darwin":
		sh := `if [ -d "$HOME/Applications/Instally.app" ]; then /System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister -f "$HOME/Applications/Instally.app"; fi`
		return []CommandSpec{{Title: "Register macOS app bundle", Shell: sh}}
	default:
		mimes := []string{"application/vnd.debian.binary-package", "application/x-deb", "application/x-rpm", "application/x-redhat-package-manager", "application/x-instally-appimage", "application/vnd.flatpak.ref", "application/vnd.flatpak.repo", "application/x-instally-arch-package", "application/zip", "application/x-tar", "application/gzip", "application/x-xz"}
		parts := []string{}
		for _, m := range mimes {
			parts = append(parts, "xdg-mime default instally.desktop "+m+" || true")
		}
		return []CommandSpec{{Title: "Set Instally as default installer on Linux", Shell: strings.Join(parts, "\n")}}
	}
}

func UnsetDefaultCommands() []CommandSpec {
	if runtime.GOOS != "linux" {
		return []CommandSpec{{Title: "Unset default", Shell: "echo 'Use system settings to remove Instally association on this OS'"}}
	}
	mimes := []string{"application/vnd.debian.binary-package", "application/x-deb", "application/x-rpm", "application/x-redhat-package-manager", "application/x-instally-appimage", "application/vnd.flatpak.ref", "application/vnd.flatpak.repo", "application/x-instally-arch-package"}
	parts := []string{}
	for _, m := range mimes {
		parts = append(parts, "xdg-mime default '' "+m+" || true")
	}
	return []CommandSpec{{Title: "Unset Instally defaults", Shell: strings.Join(parts, "\n")}}
}

func FullSetupPlan(self string) Plan {
	sys := Detect()
	p := Plan{System: sys, Tasks: []Task{{Kind: "full-setup", Items: []string{self}}}}
	if sys.Manager.ID != "none" && len(sys.Manager.Prepare) > 0 {
		cmd := append([]string{}, sys.Manager.Install...)
		cmd = append(cmd, sys.Manager.Prepare...)
		cmd = append(cmd, sys.Manager.Yes...)
		p.Commands = append(p.Commands, withManagerRefresh(CommandSpec{Title: "Install common build/install dependencies", Cmd: cmd, Admin: sys.Manager.NeedsElev}, sys.Manager))
	}
	p.Commands = append(p.Commands, InstallCommands(self, true)...)
	return p
}

func linuxDesktop(bin string) string {
	return `[Desktop Entry]
Type=Application
Name=Instally
GenericName=Universal installer
Comment=Install apps from packages, stores, Git and releases
Exec=` + bin + `
Terminal=true
Icon=instally
Categories=System;PackageManager;Utility;
StartupNotify=true
`
}

func mimeXML() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<mime-info xmlns="http://www.freedesktop.org/standards/shared-mime-info">
  <mime-type type="application/x-instally-appimage"><comment>AppImage bundle</comment><glob pattern="*.AppImage"/><glob pattern="*.appimage"/></mime-type>
  <mime-type type="application/x-instally-arch-package"><comment>Arch Linux package</comment><glob pattern="*.pkg.tar.zst"/><glob pattern="*.pkg.tar.xz"/><glob pattern="*.pkg.tar.gz"/></mime-type>
</mime-info>`
}

func macPlist() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict><key>CFBundleName</key><string>Instally</string><key>CFBundleIdentifier</key><string>io.github.instally</string><key>CFBundleExecutable</key><string>Instally</string><key>CFBundlePackageType</key><string>APPL</string></dict></plist>`
}

func SelfPath() string {
	p, err := os.Executable()
	if err != nil {
		return "instally"
	}
	return p
}
