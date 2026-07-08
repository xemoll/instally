package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// RunTerminalInstaller is a plain terminal workflow for Linux shells and SSH sessions.
// It avoids GUI/browser UI: paste program names, URLs, GitHub repos or local paths,
// review the generated plan, then confirm execution. If a command needs admin rights,
// RunPlan will use pkexec when available in a desktop session, otherwise sudo so the
// user can type a password in the terminal.
func RunTerminalInstaller(opts Options) int {
	fmt.Println("Instally terminal installer")
	fmt.Println("Paste apps, URLs, GitHub repos or local files. Use commas or one item per line.")
	fmt.Println("Examples: vscode, discord, github:cli/cli, https://example.com/app.AppImage")
	fmt.Println("End input with an empty line or Ctrl+D.")
	fmt.Println()

	interactive := stdinIsTerminal()
	var lines []string
	s := bufio.NewScanner(os.Stdin)
	for {
		if interactive {
			fmt.Print("instally> ")
		}
		if !s.Scan() {
			break
		}
		line := strings.TrimSpace(s.Text())
		if line == "" {
			break
		}
		lines = append(lines, line)
	}
	if err := s.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "input error:", err)
		return 1
	}
	if len(lines) == 0 {
		fmt.Fprintln(os.Stderr, "nothing to install")
		return 1
	}

	tasks := ParseMultiItems(strings.Join(lines, "\n"))
	plan := BuildPlan(tasks, Options{Yes: true, DryRun: opts.DryRun, AllowUnknown: opts.AllowUnknown, VirusTotalKey: opts.VirusTotalKey, VirusTotalUpload: opts.VirusTotalUpload, ContinueOnError: opts.ContinueOnError})
	fmt.Println()
	fmt.Println("Plan:")
	for i, c := range plan.Commands {
		admin := ""
		if c.Admin {
			admin = " [admin]"
		}
		fmt.Printf("%2d. %s%s\n    %s\n", i+1, c.Title, admin, commandLine(c))
	}
	for _, w := range plan.Warnings {
		fmt.Println("warning:", w)
	}
	if opts.DryRun {
		fmt.Println("dry-run: nothing executed")
		return 0
	}
	if !opts.Yes && interactive {
		fmt.Print("Run this plan? [y/N] ")
		ans := bufio.NewScanner(os.Stdin)
		if !ans.Scan() || !strings.EqualFold(strings.TrimSpace(ans.Text()), "y") {
			fmt.Println("cancelled")
			return 1
		}
	}
	fmt.Println()
	fmt.Println("Running. If admin rights are needed, Instally will open pkexec or ask sudo password here.")
	res := RunPlan(plan, false)
	fmt.Print(res.Output)
	if !res.OK {
		return res.ExitCode
	}
	return 0
}

func stdinIsTerminal() bool {
	st, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (st.Mode() & os.ModeCharDevice) != 0
}
