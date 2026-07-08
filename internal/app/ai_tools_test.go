package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestAIToolsFlagPlanLinuxPacman(t *testing.T) {
	t.Setenv("INSTALLY_FORCE_OS", "linux")
	t.Setenv("INSTALLY_FORCE_PM", "pacman")
	p := BuildPlan([]Task{{Kind: "ai-tools", Items: []string{"ai-tools"}}}, Options{Yes: true, DryRun: true})
	line := commandLine(CommandSpec{Shell: strings.Join(func() []string {
		out := []string{}
		for _, c := range p.Commands {
			out = append(out, commandLine(c))
		}
		return out
	}(), "\n")})
	for _, want := range []string{"opencode", "ollama.com/install.sh", "claude.ai/install.sh"} {
		if !strings.Contains(line, want) {
			t.Fatalf("ai-tools pacman plan missing %s:\n%s", want, line)
		}
	}
}

func TestOpenCodeOfficialPlans(t *testing.T) {
	cases := []struct{ os, pm, want string }{{"linux", "pacman", "opencode"}, {"linux", "apt", "opencode-ai"}, {"darwin", "brew", "anomalyco/tap/opencode"}, {"windows", "scoop", "scoop install opencode"}}
	for _, tc := range cases {
		t.Run(tc.os+"-"+tc.pm, func(t *testing.T) {
			t.Setenv("INSTALLY_FORCE_OS", tc.os)
			t.Setenv("INSTALLY_FORCE_PM", tc.pm)
			p := BuildPlan([]Task{{Kind: "app", Items: []string{"opencode"}}}, Options{Yes: true, DryRun: true})
			var all string
			for _, c := range p.Commands {
				all += commandLine(c) + "\n"
			}
			if !strings.Contains(all, tc.want) {
				t.Fatalf("plan missing %q:\n%s", tc.want, all)
			}
		})
	}
}

func TestClaudeCodeOfficialManagers(t *testing.T) {
	cases := []struct{ pm, want string }{{"apt", "downloads.claude.ai/claude-code/apt/stable"}, {"dnf", "claude-code/rpm/stable"}, {"apk", "claude-code/apk/stable"}}
	for _, tc := range cases {
		t.Run(tc.pm, func(t *testing.T) {
			t.Setenv("INSTALLY_FORCE_OS", "linux")
			t.Setenv("INSTALLY_FORCE_PM", tc.pm)
			p := BuildPlan([]Task{{Kind: "app", Items: []string{"claude-code"}}}, Options{Yes: true, DryRun: true})
			var all string
			for _, c := range p.Commands {
				all += commandLine(c) + "\n"
			}
			if !strings.Contains(all, tc.want) {
				t.Fatalf("claude plan missing %q:\n%s", tc.want, all)
			}
		})
	}
}

func TestOllamaOfficialPlansByOS(t *testing.T) {
	cases := []struct{ os, pm, want string }{{"linux", "apt", "ollama.com/install.sh"}, {"windows", "winget", "Ollama.Ollama"}, {"darwin", "brew", "ollama"}}
	for _, tc := range cases {
		t.Run(tc.os+"-"+tc.pm, func(t *testing.T) {
			t.Setenv("INSTALLY_FORCE_OS", tc.os)
			t.Setenv("INSTALLY_FORCE_PM", tc.pm)
			p := BuildPlan([]Task{{Kind: "app", Items: []string{"ollama"}}}, Options{Yes: true, DryRun: true})
			var all string
			for _, c := range p.Commands {
				all += commandLine(c) + "\n"
			}
			if !strings.Contains(all, tc.want) {
				t.Fatalf("ollama plan missing %q:\n%s", tc.want, all)
			}
		})
	}
}

func TestVirusTotalMockHashUnsafe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/files/") {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("x-apikey") != "mock-key" {
			t.Fatalf("missing vt key")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"attributes":{"last_analysis_stats":{"malicious":2,"suspicious":0,"harmless":1,"undetected":4}}}}`))
	}))
	defer srv.Close()
	t.Setenv("INSTALLY_VT_API_BASE", srv.URL)
	f := writeTempFile(t, "tool.sh", "echo ok")
	rep := ScanFile(f, SecurityOptions{VirusTotalKey: "mock-key", AllowUnknown: true})
	if rep.Status != "unsafe" || !hasCheck(rep, "VirusTotal", "unsafe", "malicious=2") {
		t.Fatalf("VT mock unsafe not applied: %#v", rep)
	}
}

func TestVirusTotalMockUploadClean(t *testing.T) {
	var uploaded bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/files/") && r.Method == "GET":
			w.WriteHeader(404)
			_, _ = w.Write([]byte(`{"error":{"code":"NotFoundError","message":"not found"}}`))
		case r.URL.Path == "/files" && r.Method == "POST":
			uploaded = true
			_, _ = w.Write([]byte(`{"data":{"id":"analysis-1"}}`))
		case r.URL.Path == "/analyses/analysis-1":
			_, _ = w.Write([]byte(`{"data":{"attributes":{"status":"completed","stats":{"malicious":0,"suspicious":0,"harmless":2,"undetected":10}}}}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()
	t.Setenv("INSTALLY_VT_API_BASE", srv.URL)
	f := writeTempFile(t, "small.bin", "hello")
	rep := ScanFile(f, SecurityOptions{VirusTotalKey: "mock-key", VirusTotalUpload: true, AllowUnknown: true})
	if !uploaded || !hasCheck(rep, "VirusTotal", "clean", "uploaded analysis") {
		t.Fatalf("VT upload clean missing uploaded=%v checks=%#v", uploaded, rep.Checks)
	}
}

func TestOfficialInstallerDoesNotLeakVTKey(t *testing.T) {
	p := BuildPlan([]Task{{Kind: "app", Items: []string{"ollama"}}}, Options{Yes: true, DryRun: true, VirusTotalKey: "super-secret-token", VirusTotalUpload: true})
	var all string
	for _, c := range p.Commands {
		all += commandLine(c) + "\n"
	}
	if strings.Contains(all, "super-secret-token") || strings.Contains(all, "INSTALLY_VT_API_KEY") {
		t.Fatalf("VT key leaked into child command/env:\n%s", all)
	}
}

func TestStaticSevereScriptBlocks(t *testing.T) {
	f := writeTempFile(t, "wipe.sh", "#!/bin/sh\nrm -rf / --no-preserve-root\n")
	rep := ScanFile(f, SecurityOptions{AllowUnknown: true})
	if rep.Status != "unsafe" || !hasCheck(rep, "Статический анализ", "unsafe", "rm -rf /") {
		t.Fatalf("severe script was not blocked: %#v", rep)
	}
}

func TestAIAppAliases(t *testing.T) {
	for _, name := range []string{"opencode", "open-code", "claude", "claude-code", "ai-tools"} {
		if task := AutoTask(name); task.Kind != "app" {
			t.Fatalf("%s should be app alias: %#v", name, task)
		}
	}
}

func TestForbiddenSecretFixture(t *testing.T) {
	secret := os.Getenv("INSTALLY_FORBIDDEN_TEST_SECRET")
	if secret == "" {
		t.Skip("set INSTALLY_FORBIDDEN_TEST_SECRET to scan for a concrete secret")
	}
	if strings.Contains(secret, " ") {
		t.Fatal("bad secret fixture")
	}
}
