package app

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func RunTUI(opts Options) int {
	sys := Detect()
	app := tview.NewApplication()
	st := &tuiState{app: app, opts: opts}

	input := tview.NewInputField()
	input.SetPlaceholder("firefox, discord, github:cli/cli, https://...")
	input.SetFieldWidth(0)
	input.SetLabel("[green]instally[white] > ")

	out := tview.NewTextView()
	out.SetDynamicColors(true)
	out.SetScrollable(true)
	out.SetWordWrap(true)
	out.SetText("\n  напиши что установить")

	updateNote := ""
	updateHint := ""
	ui := SelfUpdateCheck()
	if ui.Available {
		updateNote = fmt.Sprintf(" · [green]↑ v%s[white]", ui.Latest)
		updateHint = "  ^u обновить"
	}

	bar := tview.NewTextView()
	bar.SetDynamicColors(true)
	bar.SetText(fmt.Sprintf("[gray]%s  ⏎ авто%s ^e сканировать  ^r установить  ^s поиск  esc выход[white]", SystemLabelForUI(sys), updateHint))

	top := tview.NewTextView()
	top.SetDynamicColors(true)
	top.SetTextAlign(tview.AlignCenter)
	top.SetText(fmt.Sprintf("[gray]instally · %s%s[white]", SystemLabelForUI(sys), updateNote))

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(top, 1, 0, false).
		AddItem(input, 1, 0, true).
		AddItem(out, 0, 1, false).
		AddItem(bar, 1, 0, false)

	input.SetChangedFunc(func(text string) {
		st.showInput(text, out)
	})
	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			st.showInput(strings.TrimSpace(input.GetText()), out)
		}
	})

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch {
		case event.Key() == tcell.KeyCtrlE:
			st.showScan(strings.TrimSpace(input.GetText()), out)
			return nil
		case event.Key() == tcell.KeyCtrlR:
			st.showInstall(strings.TrimSpace(input.GetText()), out, app)
			return nil
		case event.Key() == tcell.KeyCtrlS:
			st.showSearch(strings.TrimSpace(input.GetText()), out)
			return nil
		case event.Key() == tcell.KeyCtrlU:
			go st.showUpdate(out, app)
			return nil
		case event.Key() == tcell.KeyEscape:
			app.Stop()
			return nil
		}
		return event
	})

	if err := app.SetRoot(flex, true).EnableMouse(false).Run(); err != nil {
		return 2
	}
	return 0
}

type tuiState struct {
	app  *tview.Application
	opts Options
	scan *ScanInputResult
	last string
}

func (st *tuiState) showInput(text string, out *tview.TextView) {
	if text == "" || text == st.last {
		if text == "" {
			out.SetText("\n  напиши что установить")
			st.last = ""
			return
		}
		return
	}
	st.last = text

	tasks := ParseBatchText(text)
	if len(tasks) == 0 {
		out.SetText("[yellow]  ничего не распознано[white]")
		return
	}

	var lines []string
	for _, task := range tasks {
		for _, item := range task.Items {
			lines = append(lines, fmt.Sprintf("  %s [white]%s[gray]  %s", srcIcon(task.Kind), item, SourceKindFriendly(task.Kind)))
		}
	}

	plan := BuildPlan(tasks, Options{Yes: true, DryRun: true, NoSecurity: true})
	if s := PlanSummaryForUI(plan); s != "" {
		lines = append(lines, fmt.Sprintf("\n  [green]%s[white]", s))
	}
	for i, cmd := range plan.Commands {
		if i >= 5 {
			lines = append(lines, fmt.Sprintf("  [gray]… +%d[white]", len(plan.Commands)-5))
			break
		}
		adm := ""
		if cmd.Admin {
			adm = " [red]🔒[white]"
		}
		cl := commandLine(cmd)
		if len(cl) > 55 {
			cl = cl[:52] + "..."
		}
		lines = append(lines, fmt.Sprintf("  [gray]%d.[white] %s%s", i+1, cmd.Title, adm))
	}
	for _, w := range plan.Warnings {
		lines = append(lines, fmt.Sprintf("  [yellow]! %s[white]", w))
	}

	out.SetText(strings.Join(lines, "\n"))
}

func (st *tuiState) showScan(text string, out *tview.TextView) {
	if text == "" {
		return
	}

	sec := SecurityOptionsFromEnv()
	res := ScanInputText(text, sec)
	st.scan = &res

	var lines []string
	for _, t := range res.Targets {
		rep := t.Report
		src := t.Source
		if src == "" {
			src = t.Item
		}
		lines = append(lines, fmt.Sprintf("  %s [::b]%s[::-]", stIcon(rep.Status), src))
		lines = append(lines, fmt.Sprintf("    [%s]%s[white]", stColor(rep.Status), SecurityStatusFriendly(rep.Status)))
		lines = append(lines, fmt.Sprintf("    %s", SecurityHumanSummaryForUI(rep)))
		for _, c := range rep.Checks {
			lines = append(lines, fmt.Sprintf("    %s %s", stIcon(c.Status), c.Detail))
		}
	}
	for _, w := range res.Warnings {
		lines = append(lines, fmt.Sprintf("  [yellow]! %s[white]", w))
	}
	if res.Safe {
		lines = append(lines, "\n  [green]✓ чисто[white]")
	} else {
		lines = append(lines, "\n  [yellow]! требуется подтверждение[white]")
	}
	out.SetText(strings.Join(lines, "\n"))
}

func (st *tuiState) showSearch(text string, out *tview.TextView) {
	if text == "" {
		out.SetText("[yellow]  введи запрос и нажми ^s[white]")
		return
	}
	r := SearchPackages(text)
	if strings.TrimSpace(r) == "" {
		r = "[yellow]  ничего не найдено[white]"
	}
	out.SetText(r)
}

func (st *tuiState) showInstall(text string, out *tview.TextView, a *tview.Application) {
	if text == "" {
		return
	}

	lo := tview.NewTextView()
	lo.SetDynamicColors(true)
	lo.SetScrollable(true)
	lo.SetWordWrap(true)

	loBar := tview.NewTextView()
	loBar.SetDynamicColors(true)
	loBar.SetTextAlign(tview.AlignCenter)
	loBar.SetText("[gray]^c отмена[white]")

	a.SetRoot(tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(lo, 0, 1, false).
		AddItem(loBar, 1, 0, false), true)

	go func() {
		b := &tuiBuf{out: lo, app: a}
		b.write("подготовка...\n")

		sec := SecurityOptionsFromEnv()
		res := st.scan
		if res == nil {
			r := ScanInputText(text, sec)
			res = &r
		}

		for _, t := range res.Targets {
			src := t.Source
			if src == "" {
				src = t.Item
			}
			b.write(fmt.Sprintf("источник: %s\n  %s\n", src, SecurityHumanSummaryForUI(t.Report)))
		}

		if !res.Safe {
			b.write("\n[yellow]проверка не пройдена[white]\n")
			return
		}

		b.write("установка...\n")
		secOpts := SecurityOptionsFromEnv()
		installOpts := Options{
			Yes: true,
			NoSecurity: false,
			VirusTotalKey: secOpts.VirusTotalKey,
			VirusTotalUpload: secOpts.VirusTotalUpload,
			AllowUnknown: secOpts.AllowUnknown,
		}
		plan := BuildPlan(TasksForCheckedInstall(*res, ParseBatchText(text)), installOpts)
		r := RunPlan(plan, false)
		if r.Output != "" {
			b.write(r.Output)
		}
		if r.OK {
			b.write("\n[green]✓ готово[white]\n")
		} else {
			b.write("\n[red]✗ ошибка[white]\n")
		}
	}()

	a.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlC {
			a.Stop()
			return nil
		}
		return event
	})
}

type tuiBuf struct {
	out *tview.TextView
	app *tview.Application
	mu  sync.Mutex
	buf strings.Builder
}

func (b *tuiBuf) write(s string) {
	b.mu.Lock()
	b.buf.WriteString(s)
	b.mu.Unlock()
	b.app.QueueUpdateDraw(func() {
		b.out.SetText(b.buf.String())
		b.out.ScrollToEnd()
	})
}

func (st *tuiState) showUpdate(out *tview.TextView, a *tview.Application) {
	ui := SelfUpdateCheck()
	if ui.Error != "" {
		out.SetText(fmt.Sprintf("[red]update check: %s[white]", ui.Error))
		return
	}
	if !ui.Available {
		out.SetText(fmt.Sprintf("[green]up to date (v%s)[white]", ui.Current))
		return
	}
	b := &tuiBuf{out: out, app: a}
	b.write(fmt.Sprintf("Update available: v%s → v%s\n", ui.Current, ui.Latest))
	b.write(fmt.Sprintf("Asset: %s (%s)\n", ui.AssetName, humanSize(ui.Size)))
	b.write("\nrun outside TUI: instally --update-self\n")
}

func srcIcon(kind string) string {
	switch kind {
	case "url":
		return "[blue]↓[white]"
	case "github", "release":
		return "[purple]◆[white]"
	case "git":
		return "[darkcyan]○[white]"
	case "local":
		return "[yellow]f[white]"
	case "app", "pkg":
		return "[green]●[white]"
	case "flatpak":
		return "[blue]⬡[white]"
	default:
		return "[gray]·[white]"
	}
}

func stIcon(s string) string {
	switch s {
	case "clean":
		return "[green]✓[white]"
	case "unsafe":
		return "[red]✗[white]"
	case "error":
		return "[red]![white]"
	case "limited", "warning":
		return "[yellow]◐[white]"
	default:
		return "[gray]·[white]"
	}
}

func stColor(s string) string {
	switch s {
	case "clean":
		return "green"
	case "unsafe", "error":
		return "red"
	case "limited", "warning":
		return "yellow"
	default:
		return "gray"
	}
}
