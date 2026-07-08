package app

import (
    "strings"
    "testing"
)

func TestCommandTimeoutEnv(t *testing.T) {
    t.Setenv("INSTALLY_COMMAND_TIMEOUT_SECONDS", "1")
    code, text, err := runCommand(CommandSpec{Title: "slow", Shell: "sleep 3"})
    if err == nil || code != 124 {
        t.Fatalf("expected timeout code 124, got code=%d err=%v text=%q", code, err, text)
    }
    if !strings.Contains(err.Error(), "timed out") {
        t.Fatalf("timeout error should be clear, got %v", err)
    }
}

func TestRefreshTimeoutEnv(t *testing.T) {
    t.Setenv("INSTALLY_REFRESH_TIMEOUT_SECONDS", "1")
    code, text, err := runCommand(CommandSpec{Title: "fail then slow refresh", Cmd: []string{"sh", "-c", "exit 7"}, Refresh: []string{"sh", "-c", "sleep 3"}})
    if err == nil || code != 124 {
        t.Fatalf("expected refresh timeout code 124, got code=%d err=%v text=%q", code, err, text)
    }
    if !strings.Contains(text, "обновляю метаданные") {
        t.Fatalf("expected refresh message, got %q", text)
    }
}

func TestTimeoutDiagnosticText(t *testing.T) {
    diag := diagnoseCommandFailure(CommandSpec{Cmd: []string{"apt-get", "install", "missing"}, Refresh: []string{"apt-get", "update"}}, "command timed out after 1s")
    if !strings.Contains(diag, "timed out") && !strings.Contains(diag, "DNS") {
        t.Fatalf("expected timeout/DNS diagnostic, got %q", diag)
    }
}
