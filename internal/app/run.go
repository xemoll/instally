package app

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func RunPlan(plan Plan, dry bool) RunResult {
	res := RunResult{DryRun: dry, OK: true, Plan: plan}
	var out bytes.Buffer
	if len(plan.Commands) == 0 && len(plan.Warnings) > 0 {
		res.OK = false
		res.ExitCode = 2
		res.Errors = append(res.Errors, "no runnable commands; see warnings")
	}
	for i, c := range plan.Commands {
		line := commandLine(c)
		fmt.Fprintf(&out, "[%d/%d] %s\n%s\n", i+1, len(plan.Commands), c.Title, line)
		if hint := refreshLine(c); hint != "" {
			fmt.Fprintf(&out, "%s\n", hint)
		}
		if dry {
			continue
		}
		code, text, err := runCommand(c)
		out.WriteString(text)
		if err != nil || code != 0 {
			res.OK = false
			res.ExitCode = code
			if err != nil {
				res.Errors = append(res.Errors, err.Error())
				fmt.Fprintf(&out, "error: %s\n", err)
				if text == "" {
					text = err.Error()
				}
			}
			if diag := diagnoseCommandFailure(c, text); diag != "" {
				res.Errors = append(res.Errors, diag)
				fmt.Fprintf(&out, "diagnostic: %s\n", diag)
			}
			if !planContinueOnError(plan) {
				break
			}
			fmt.Fprintf(&out, "warning: command failed, continuing because continue-on-error is enabled\n")
		}
	}
	for _, w := range plan.Warnings {
		fmt.Fprintf(&out, "warning: %s\n", w)
	}
	res.Output = out.String()
	return res
}

func RunPlanStream(plan Plan, dry bool, w io.Writer) RunResult {
	res := RunResult{DryRun: dry, OK: true, Plan: plan}
	if len(plan.Commands) == 0 && len(plan.Warnings) > 0 {
		res.OK = false
		res.ExitCode = 2
		res.Errors = append(res.Errors, "no runnable commands; see warnings")
	}
	write := func(format string, args ...any) {
		fmt.Fprintf(w, format, args...)
		if f, ok := w.(interface{ Flush() }); ok {
			f.Flush()
		}
	}
	for i, c := range plan.Commands {
		line := commandLine(c)
		write("\n[%d/%d] %s\n%s\n", i+1, len(plan.Commands), c.Title, line)
		if hint := refreshLine(c); hint != "" {
			write("%s\n", hint)
		}
		if dry {
			continue
		}
		code, err := runCommandStream(c, w)
		if err != nil || code != 0 {
			res.OK = false
			res.ExitCode = code
			if err != nil {
				res.Errors = append(res.Errors, err.Error())
				write("error: %s\n", err)
			}
			if diag := diagnoseCommandFailure(c, ""); diag != "" {
				res.Errors = append(res.Errors, diag)
				write("diagnostic: %s\n", diag)
			}
			if !planContinueOnError(plan) {
				break
			}
			write("warning: command failed, continuing because continue-on-error is enabled\n")
		}
	}
	for _, warn := range plan.Warnings {
		write("warning: %s\n", warn)
	}
	if len(plan.Commands) == 0 && len(plan.Warnings) > 0 {
		write("error: no runnable commands; see warnings\n")
	}
	if res.OK {
		write("\nInstally: готово\n")
	}
	return res
}

func planContinueOnError(plan Plan) bool { return plan.ContinueOnError }

func commandTimeout(c CommandSpec) time.Duration {
	if c.TimeoutSeconds > 0 {
		return time.Duration(c.TimeoutSeconds) * time.Second
	}
	if v := strings.TrimSpace(os.Getenv("INSTALLY_COMMAND_TIMEOUT_SECONDS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return 6 * time.Hour
}

func refreshTimeoutSeconds() int {
	if v := strings.TrimSpace(os.Getenv("INSTALLY_REFRESH_TIMEOUT_SECONDS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 120
}

func shellCommandContext(ctx context.Context, shell string, admin bool) (*exec.Cmd, error) {
	if runtime.GOOS == "windows" {
		if admin {
			ps := "Start-Process -Verb RunAs -Wait -FilePath cmd.exe -ArgumentList '/C'," + winPSQuote(shell)
			return exec.CommandContext(ctx, "powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", ps), nil
		}
		return exec.CommandContext(ctx, "cmd", "/C", shell), nil
	}
	if admin && !isRoot() {
		if commandExists("pkexec") != "" && os.Getenv("DISPLAY") != "" {
			return exec.CommandContext(ctx, "pkexec", "/bin/sh", "-lc", shell), nil
		}
		if commandExists("sudo") != "" {
			return exec.CommandContext(ctx, "sudo", "/bin/sh", "-lc", shell), nil
		}
		return nil, fmt.Errorf("admin rights required, but neither pkexec nor sudo is available")
	}
	return exec.CommandContext(ctx, "/bin/sh", "-lc", shell), nil
}

func runCommandStream(c CommandSpec, w io.Writer) (int, error) {
	code, err := runCommandStreamNoRetry(c, w)
	if (err == nil && code == 0) || len(c.Refresh) == 0 {
		return code, err
	}
	fmt.Fprintf(w, "\nInstally: команда не прошла; обновляю метаданные менеджера пакетов: %s\n", shellJoin(c.Refresh))
	refresh := CommandSpec{Title: "Refresh package metadata", Cmd: c.Refresh, Admin: c.Admin, Env: c.Env, TimeoutSeconds: refreshTimeoutSeconds()}
	refreshCode, refreshErr := runCommandStreamNoRetry(refresh, w)
	if refreshErr != nil || refreshCode != 0 {
		fmt.Fprintf(w, "Instally: refresh не помог завершиться успешно. %s\n", diagnoseCommandFailure(c, "refresh failed"))
		return refreshCode, refreshErr
	}
	fmt.Fprintf(w, "Instally: повторяю исходную команду один раз\n")
	retry := c
	retry.Refresh = nil
	code, err = runCommandStreamNoRetry(retry, w)
	if err != nil || code != 0 {
		if diag := diagnoseCommandFailure(c, "retry failed"); diag != "" {
			fmt.Fprintf(w, "diagnostic: %s\n", diag)
		}
	}
	return code, err
}

func cleanupSecretFiles(env map[string]string) {
	if len(env) == 0 {
		return
	}
	paths := []string{}
	if p := strings.TrimSpace(env["INSTALLY_VT_KEY_FILE"]); p != "" {
		paths = append(paths, p)
	}
	if p := strings.TrimSpace(env["INSTALLY_SECRET_CLEANUP"]); p != "" {
		for _, item := range strings.Split(p, string(os.PathListSeparator)) {
			if strings.TrimSpace(item) != "" {
				paths = append(paths, strings.TrimSpace(item))
			}
		}
	}
	for _, p := range appendUnique(nil, paths...) {
		_ = os.Remove(p)
	}
}

func runCommandStreamNoRetry(c CommandSpec, w io.Writer) (int, error) {
	defer cleanupSecretFiles(c.Env)
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout(c))
	defer cancel()
	var cmd *exec.Cmd
	if c.Shell != "" {
		var err error
		cmd, err = shellCommandContext(ctx, c.Shell, c.Admin)
		if err != nil {
			return 126, err
		}
	} else if len(c.Cmd) > 0 {
		args := append([]string{}, c.Cmd...)
		prog := args[0]
		args = args[1:]
		if c.Admin && runtime.GOOS != "windows" && !isRoot() {
			if commandExists("pkexec") != "" && os.Getenv("DISPLAY") != "" {
				args = append([]string{prog}, args...)
				prog = "pkexec"
			} else if commandExists("sudo") != "" {
				args = append([]string{prog}, args...)
				prog = "sudo"
			} else {
				return 126, fmt.Errorf("admin rights required for %s, but neither pkexec nor sudo is available", c.Title)
			}
		}
		cmd = exec.CommandContext(ctx, prog, args...)
	} else {
		return 0, nil
	}
	if c.Dir != "" {
		cmd.Dir = c.Dir
	}
	if len(c.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range c.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	if c.Admin || c.Shell != "" {
		cmd.Stdin = os.Stdin
	}
	cmd.Stdout = w
	cmd.Stderr = w
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return 124, fmt.Errorf("command timed out after %s: %s", commandTimeout(c), commandLine(c))
	}
	code := 0
	if err != nil {
		code = 1
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		}
	}
	return code, err
}

func runCommand(c CommandSpec) (int, string, error) {
	code, text, err := runCommandNoRetry(c)
	if (err == nil && code == 0) || len(c.Refresh) == 0 {
		return code, text, err
	}
	var out bytes.Buffer
	out.WriteString(text)
	fmt.Fprintf(&out, "\nInstally: команда не прошла; обновляю метаданные менеджера пакетов: %s\n", shellJoin(c.Refresh))
	refresh := CommandSpec{Title: "Refresh package metadata", Cmd: c.Refresh, Admin: c.Admin, Env: c.Env, TimeoutSeconds: refreshTimeoutSeconds()}
	refreshCode, refreshText, refreshErr := runCommandNoRetry(refresh)
	out.WriteString(refreshText)
	if refreshErr != nil || refreshCode != 0 {
		if diag := diagnoseCommandFailure(c, out.String()); diag != "" {
			fmt.Fprintf(&out, "diagnostic: %s\n", diag)
		}
		return refreshCode, out.String(), refreshErr
	}
	fmt.Fprintf(&out, "Instally: повторяю исходную команду один раз\n")
	retry := c
	retry.Refresh = nil
	code, retryText, err := runCommandNoRetry(retry)
	out.WriteString(retryText)
	if err != nil || code != 0 {
		if diag := diagnoseCommandFailure(c, out.String()); diag != "" {
			fmt.Fprintf(&out, "diagnostic: %s\n", diag)
		}
	}
	return code, out.String(), err
}

func runCommandNoRetry(c CommandSpec) (int, string, error) {
	defer cleanupSecretFiles(c.Env)
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout(c))
	defer cancel()
	var cmd *exec.Cmd
	if c.Shell != "" {
		var err error
		cmd, err = shellCommandContext(ctx, c.Shell, c.Admin)
		if err != nil {
			return 126, "", err
		}
	} else if len(c.Cmd) > 0 {
		args := append([]string{}, c.Cmd...)
		prog := args[0]
		args = args[1:]
		if c.Admin && runtime.GOOS != "windows" && !isRoot() {
			if commandExists("pkexec") != "" && os.Getenv("DISPLAY") != "" {
				args = append([]string{prog}, args...)
				prog = "pkexec"
			} else if commandExists("sudo") != "" {
				args = append([]string{prog}, args...)
				prog = "sudo"
			} else {
				return 126, "", fmt.Errorf("admin rights required for %s, but neither pkexec nor sudo is available", c.Title)
			}
		}
		cmd = exec.CommandContext(ctx, prog, args...)
	} else {
		return 0, "", nil
	}
	if c.Dir != "" {
		cmd.Dir = c.Dir
	}
	if len(c.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range c.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	var out, errBuf bytes.Buffer
	if c.Admin || c.Shell != "" {
		// Keep stdin attached so sudo/pkexec/script installers can ask for input,
		// but capture stdout/stderr here. The streaming runner is responsible for
		// live display; the buffered runner must not print twice or out of order.
		cmd.Stdin = os.Stdin
	}
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return 124, out.String(), fmt.Errorf("command timed out after %s: %s", commandTimeout(c), commandLine(c))
	}
	code := 0
	if err != nil {
		code = 1
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		}
		if errBuf.Len() > 0 {
			err = fmt.Errorf("%w\nstderr:\n%s", err, strings.TrimSpace(errBuf.String()))
		}
	}
	return code, out.String(), err
}

func SupportSummary() string {
	sys := Detect()
	items := []struct {
		Name string
		OK   bool
		Hint string
	}{
		{"Пакеты системы", sys.Manager.ID != "none", "нужен apt/pacman/dnf/zypper/apk/xbps/eopkg/brew/winget/scoop/choco или PackageKit"},
		{"Git/GitHub", sys.Tools["git"] != "", "установи git"},
		{"Скачивание URL", true, "встроенный Go downloader работает без curl/wget"},
		{"Flatpak", sys.Tools["flatpak"] != "", "Instally попробует поставить flatpak через системный менеджер"},
		{"Snap", sys.Tools["snap"] != "", "Instally попробует поставить snapd там, где это поддерживается"},
		{"npm", sys.Tools["npm"] != "" || sys.Tools["node"] != "", "Instally умеет поставить node/npm через менеджер"},
		{"Cargo", sys.Tools["cargo"] != "", "Instally умеет поставить cargo/rust через менеджер"},
		{"Go install", sys.Tools["go"] != "", "Instally умеет поставить go через менеджер"},
		{"VirusTotal", strings.TrimSpace(SecurityOptionsFromEnv().VirusTotalKey) != "", "не обязателен; без ключа используются локальные проверки, с ключом — hash lookup/upload"},
		{"Локальный сканер", sys.Tools["clamscan"] != "" || sys.Tools["clamdscan"] != "" || sys.Tools["powershell"] != "" || sys.Tools["pwsh"] != "", "ClamAV на Linux/macOS или Defender на Windows"},
		{"YARA", sys.Tools["yara"] != "", "дополнительные сигнатуры; Instally работает и без него"},
		{"Archive safety", true, "встроенная проверка zip/tar path-traversal, symlink и archive-bomb признаков"},
		{"Static heuristics", true, "поиск опасных скриптовых шаблонов, двойных расширений, EICAR и подозрительных структур"},
		{"Batch install", true, "--multi позволяет установить несколько приложений одной командой (vscode,discord)"},
		{"Terminal mode", true, "--terminal-install позволяет вставить список программ в терминале; sudo спросит пароль при необходимости"},
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Support matrix\n")
	fmt.Fprintf(&b, "Language: %s\n", AppLanguage())
	fmt.Fprintf(&b, "%s", VirusTotalStatus())
	for _, it := range items {
		mark := "ok"
		if !it.OK {
			mark = "missing"
		}
		fmt.Fprintf(&b, "%-16s %s — %s\n", it.Name, mark, it.Hint)
	}
	return b.String()
}

func Doctor() string {
	sys := Detect()
	var b strings.Builder
	fmt.Fprintf(&b, "Instally doctor\n")
	fmt.Fprintf(&b, "OS: %s (%s/%s)\n", sys.Family, sys.GOOS, sys.Arch)
	fmt.Fprintf(&b, "Distro: %s like=%s\n", sys.OSID, sys.OSLike)
	fmt.Fprintf(&b, "Manager: %s — %s\n", sys.Manager.ID, sys.Manager.Label)
	if sys.ManagerFound {
		fmt.Fprintf(&b, "Manager path: %s\n", sys.ToolPath)
	} else {
		fmt.Fprintf(&b, "Manager path: not found\n")
	}
	fmt.Fprintf(&b, "Build dir: %s\n", sys.BuildDir)
	fmt.Fprintf(&b, "Data dir: %s\n", sys.DataDir)
	fmt.Fprintf(&b, "Cache dir: %s\n", sys.CacheDir)
	for _, k := range sortedKeys(sys.Tools) {
		fmt.Fprintf(&b, "tool %-24s %s\n", k, sys.Tools[k])
	}
	ui := SelfUpdateCheck()
	if ui.Error != "" {
		fmt.Fprintf(&b, "\nUpdate check: %s\n", ui.Error)
	} else if ui.Available {
		fmt.Fprintf(&b, "\nUpdate available: v%s → v%s\n", ui.Current, ui.Latest)
		fmt.Fprintf(&b, "Run: instally --update-self\n")
	} else {
		fmt.Fprintf(&b, "\nUp to date (v%s)\n", ui.Current)
	}
	b.WriteString("\n")
	b.WriteString(SupportSummary())
	return b.String()
}
