package app

import (
	"fmt"
	"io"
)

func securityOptionsFromInstallOptions(opts Options) SecurityOptions {
	sec := SecurityOptionsFromEnv()
	if opts.VirusTotalKey != "" {
		sec.VirusTotalKey = opts.VirusTotalKey
	}
	if opts.VirusTotalUpload {
		sec.VirusTotalUpload = true
	}
	if opts.AllowUnknown {
		sec.AllowUnknown = true
	}
	return sec
}

func SafeRunText(text string, opts Options, w io.Writer) RunResult {
	if w == nil {
		w = io.Discard
	}
	fmt.Fprintln(w, "Instally: сначала проверяем источник")
	scan := ScanInputText(text, securityOptionsFromInstallOptions(opts))
	for _, target := range scan.Targets {
		fmt.Fprintf(w, "\nИсточник: %s\n", target.Source)
		writeSecurityHuman(w, target.Report)
	}
	for _, warn := range scan.Warnings {
		fmt.Fprintf(w, "warning: %s\n", warn)
	}
	if !scan.Safe && !opts.AllowUnknown {
		fmt.Fprintln(w, "\nInstally: установка остановлена — проверка не завершена или нашла риск")
		return RunResult{DryRun: opts.DryRun, OK: false, ExitCode: 2, Output: "security check blocked installation", Errors: []string{"security check blocked installation"}}
	}
	fmt.Fprintln(w, "\nInstally: проверка пройдена, устанавливаем из проверенного cache")
	installOpts := opts
	installOpts.NoSecurity = true
	plan := BuildPlan(TasksForCheckedInstall(scan, ParseBatchText(text)), installOpts)
	if opts.DryRun {
		res := RunPlan(plan, true)
		if res.Output != "" {
			fmt.Fprint(w, res.Output)
		}
		return res
	}
	res := RunPlan(plan, false)
	if res.Output != "" {
		fmt.Fprint(w, res.Output)
	}
	return res
}

func HumanSizeForUI(n int64) string { return humanSize(n) }

func SecurityStatusFriendly(status string) string {
	switch status {
	case "clean":
		return "Можно устанавливать"
	case "unsafe":
		return "Опасно"
	case "error":
		return "Ошибка"
	case "limited", "warning":
		return "Нужна ручная оценка"
	default:
		return "Ожидание"
	}
}
