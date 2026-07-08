package app

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUniversalKnownApps15PlusAcrossPlatforms(t *testing.T) {
	apps := []string{"vscode", "discord", "telegram", "firefox", "brave", "obs", "vlc", "blender", "gimp", "krita", "steam", "docker", "node", "go", "rust", "ollama", "opencode", "claude-code", "dbeaver", "tailscale"}
	platforms := []struct{ os, pm string }{{"linux", "pacman"}, {"linux", "apt"}, {"windows", "winget"}, {"darwin", "brew"}}
	for _, plat := range platforms {
		for _, name := range apps {
			t.Run(plat.os+"-"+plat.pm+"-"+name, func(t *testing.T) {
				t.Setenv("INSTALLY_FORCE_OS", plat.os)
				t.Setenv("INSTALLY_FORCE_PM", plat.pm)
				p := BuildPlan([]Task{AutoTask(name)}, Options{Yes: true, DryRun: true, AllowUnknown: true})
				if len(p.Commands) == 0 && len(p.Warnings) == 0 {
					t.Fatalf("no plan and no warning for %s on %s/%s", name, plat.os, plat.pm)
				}
				var all string
				for _, c := range p.Commands {
					all += commandLine(c) + "\n"
				}
				if all == "" {
					t.Fatalf("empty command plan for %s on %s/%s warnings=%v", name, plat.os, plat.pm, p.Warnings)
				}
			})
		}
	}
}

func TestMultiInstall15AppsBuildsGroupedPlan(t *testing.T) {
	t.Setenv("INSTALLY_FORCE_OS", "linux")
	t.Setenv("INSTALLY_FORCE_PM", "pacman")
	input := "vscode, discord, telegram, firefox, brave, obs, vlc, blender, gimp, krita, steam, docker, node, go, rust, ollama"
	tasks := ParseMultiItems(input)
	if len(tasks) == 0 {
		t.Fatalf("expected tasks, got none")
	}
	p := BuildPlan(tasks, Options{Yes: true, DryRun: true, AllowUnknown: true, ContinueOnError: true})
	if len(p.Commands) < 8 {
		t.Fatalf("too few commands for multi install: %d %#v", len(p.Commands), p.Commands)
	}
	var all string
	for _, c := range p.Commands {
		all += commandLine(c) + "\n"
	}
	for _, want := range []string{"flatpak", "pacman", "ollama.com/install.sh"} {
		if !strings.Contains(all, want) {
			t.Fatalf("multi plan missing %s:\n%s", want, all)
		}
	}
}

func TestInstallPresets(t *testing.T) {
	t.Setenv("INSTALLY_FORCE_OS", "linux")
	t.Setenv("INSTALLY_FORCE_PM", "apt")
	for _, preset := range []string{"base", "dev", "gaming", "media", "work", "ai", "security", "terminals"} {
		t.Run(preset, func(t *testing.T) {
			p := BuildPlan([]Task{{Kind: "preset", Items: []string{preset}}}, Options{Yes: true, DryRun: true, AllowUnknown: true})
			if len(p.Commands) == 0 {
				t.Fatalf("preset produced no commands: %s", preset)
			}
		})
	}
}

func TestFilenamePolicyBidiAndDoubleExtension(t *testing.T) {
	p := writeTempFile(t, "report.pdf.exe", "MZ fake")
	rep := ScanFile(p, SecurityOptions{AllowUnknown: true})
	if !hasCheck(rep, "Имя файла", "warning", "двойное") {
		t.Fatalf("double extension not detected: %#v", rep.Checks)
	}
	p2 := writeTempFile(t, "setup\u202etxt.exe", "MZ fake")
	rep2 := ScanFile(p2, SecurityOptions{AllowUnknown: true})
	if !hasCheck(rep2, "Имя файла", "warning", "bidi") {
		t.Fatalf("bidi filename not detected: %#v", rep2.Checks)
	}
}

func TestInstallerMetadataWarnings(t *testing.T) {
	p := writeTempFile(t, "fake.AppImage", "not an elf")
	rep := ScanFile(p, SecurityOptions{AllowUnknown: true})
	if !hasCheck(rep, "Структура файла", "warning", "AppImage") {
		t.Fatalf("fake appimage not detected: %#v", rep.Checks)
	}
}

func TestZipManyFilesBlocks(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "many.zip")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for i := 0; i < 20002; i++ {
		w, _ := zw.Create("f" + string(rune('a'+(i%26))) + "/x")
		_, _ = w.Write([]byte("x"))
	}
	_ = zw.Close()
	_ = f.Close()
	rep := ScanFile(p, SecurityOptions{AllowUnknown: true})
	if !hasCheck(rep, "Архив", "unsafe", "слишком много") {
		t.Fatalf("many files unsafe block missing: %#v", rep.Checks)
	}
}
