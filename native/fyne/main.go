package main

import (
	"bytes"
	"fmt"
	core "instally/internal/app"
	"strings"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type uiState struct {
	window       fyne.Window
	input        *widget.Entry
	status       *widget.Label
	sourceLine   *widget.Label
	steps        []*widget.Label
	resultCard   *widget.Card
	resultTitle  *widget.Label
	resultText   *widget.Label
	resultBadge  *widget.Label
	meta         *widget.Label
	checks       *fyne.Container
	planText     *widget.Label
	log          *widget.Entry
	progress     *widget.ProgressBarInfinite
	vtKey        *widget.Entry
	vtUpload     *widget.Check
	allowUnknown *widget.Check
	autoBtn      *widget.Button
	fileBtn      *widget.Button
	lastScan     core.ScanInputResult
	lastText     string
}

func main() {
	a := fyneapp.NewWithID("dev.instally.native")
	a.Settings().SetTheme(theme.DarkTheme())
	w := a.NewWindow("Instally")
	w.Resize(fyne.NewSize(1100, 820))
	w.SetMaster()

	ui := &uiState{window: w}
	ui.build(w)
	w.ShowAndRun()
}

func (ui *uiState) build(w fyne.Window) {
	sys := core.Detect()

	brand := widget.NewLabelWithStyle(core.T("app.name"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	statusPill := widget.NewLabelWithStyle("● "+core.SystemLabelForUI(sys), fyne.TextAlignTrailing, fyne.TextStyle{Monospace: true})

	ui.input = widget.NewEntry()
	ui.input.SetPlaceHolder(core.T("source.placeholder"))
	ui.input.Wrapping = fyne.TextWrapOff
	ui.input.OnChanged = func(s string) {
		ui.lastScan = core.ScanInputResult{}
		ui.resultCard.Hide()
		ui.inspect(s)
	}
	ui.input.OnSubmitted = func(string) { ui.safeInstall() }

	ui.fileBtn = widget.NewButtonWithIcon(core.T("choose.file"), theme.FileIcon(), func() { ui.chooseFile() })
	ui.fileBtn.Importance = widget.LowImportance
	ui.autoBtn = widget.NewButtonWithIcon(core.T("install.safe"), theme.DownloadIcon(), ui.safeInstall)
	ui.autoBtn.Importance = widget.HighImportance

	ui.sourceLine = widget.NewLabel(core.T("source.hint"))
	ui.status = widget.NewLabel(core.T("ready"))
	ui.progress = widget.NewProgressBarInfinite()
	ui.progress.Hide()
	ui.steps = []*widget.Label{
		widget.NewLabelWithStyle("○ Источник", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("○ Загрузка", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("○ Проверка", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("○ Установка", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
	}

	ui.resultTitle = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	ui.resultText = widget.NewLabel("")
	ui.resultBadge = widget.NewLabel("")
	ui.meta = widget.NewLabel("")
	ui.checks = container.NewVBox()
	resultBody := container.NewVBox(ui.resultTitle, ui.resultText, ui.resultBadge, ui.meta, ui.checks)
	ui.resultCard = widget.NewCard("", "", resultBody)
	ui.resultCard.Hide()

	ui.vtKey = widget.NewPasswordEntry()
	ui.vtKey.SetPlaceHolder(core.T("vt.key.placeholder"))
	ui.vtUpload = widget.NewCheck(core.T("vt.upload"), nil)
	ui.allowUnknown = widget.NewCheck(core.T("allow.limited"), nil)
	planBtn := widget.NewButton(core.T("show.plan"), ui.showPlan)
	scanOnlyBtn := widget.NewButton(core.T("scan.only"), ui.scan)
	ui.planText = widget.NewLabel(core.T("plan.placeholder"))
	langSelect := widget.NewSelect([]string{"ru", "en"}, func(v string) { core.SetAppLanguage(v); _ = core.SaveLanguage(v) })
	langSelect.SetSelected(core.AppLanguage())
	advBody := container.NewVBox(
		widget.NewLabelWithStyle(core.T("language"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		langSelect,
		widget.NewLabelWithStyle("VirusTotal", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		ui.vtKey, ui.vtUpload, ui.allowUnknown,
		container.NewGridWithColumns(2, scanOnlyBtn, planBtn),
		ui.planText,
	)
	advanced := widget.NewAccordion(widget.NewAccordionItem(core.T("advanced"), container.NewPadded(advBody)))
	ui.log = widget.NewMultiLineEntry()
	ui.log.SetPlaceHolder(core.T("log.placeholder"))
	ui.log.Wrapping = fyne.TextWrapWord
	logPane := widget.NewAccordion(widget.NewAccordionItem(core.T("log"), container.NewVScroll(ui.log)))

	w.SetContent(container.NewVBox(
		container.NewPadded(container.NewBorder(nil, nil, brand, statusPill)),
		widget.NewCard("", "", container.NewVBox(
			container.NewHBox(widget.NewIcon(theme.MailAttachmentIcon()), ui.sourceLine),
			ui.input,
			container.NewHBox(ui.fileBtn, ui.autoBtn),
		)),
		widget.NewCard("", "", container.NewVBox(
			container.NewGridWithColumns(4, ui.steps[0], ui.steps[1], ui.steps[2], ui.steps[3]),
			container.NewHBox(ui.status, ui.progress),
		)),
		ui.resultCard,
		advanced,
		logPane,
	))
	w.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
		if len(uris) == 0 {
			return
		}
		ui.input.SetText("local: " + uris[0].Path())
	})
	ui.setSteps(0)
	ui.inspect("")
}

func wrappedBold(text string) *widget.Label {
	l := widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	l.Wrapping = fyne.TextWrapWord
	return l
}

func wrappedMuted(text string) *widget.Label {
	l := widget.NewLabel(text)
	l.Wrapping = fyne.TextWrapWord
	return l
}

func stepLabel(text string) *widget.Label {
	l := widget.NewLabelWithStyle(text, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	l.Wrapping = fyne.TextWrapWord
	return l
}

func (ui *uiState) chooseFile() {
	d := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, ui.window)
			return
		}
		if r == nil {
			return
		}
		defer r.Close()
		ui.input.SetText("local: " + r.URI().Path())
	}, ui.window)
	d.Show()
}

func (ui *uiState) options() core.Options {
	return core.Options{Yes: true, VirusTotalKey: strings.TrimSpace(ui.vtKey.Text), VirusTotalUpload: ui.vtUpload.Checked, AllowUnknown: ui.allowUnknown.Checked}
}

func (ui *uiState) securityOptions() core.SecurityOptions {
	return core.SecurityOptions{VirusTotalKey: strings.TrimSpace(ui.vtKey.Text), VirusTotalUpload: ui.vtUpload.Checked, AllowUnknown: ui.allowUnknown.Checked}
}

func (ui *uiState) inspect(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		ui.sourceLine.SetText(core.T("source.hint"))
		ui.status.SetText(core.T("ready"))
		ui.setSteps(0)
		ui.planText.SetText(core.T("plan.placeholder"))
		return
	}
	info := core.InspectInputText(text)
	if len(info.Sources) == 0 {
		ui.sourceLine.SetText("Источник пока не распознан. Проверь ссылку или путь к файлу.")
		ui.status.SetText("Нужно уточнить источник")
		ui.setSteps(0)
		return
	}
	s := info.Sources[0]
	ui.sourceLine.SetText(core.SourceKindFriendly(s.Kind) + " · " + compactUI(s.Item))
	ui.status.SetText(statusForSource(s))
	ui.setSteps(stepForSource(s))
	ui.previewPlan()
}

func statusForSource(s core.SourcePreview) string {
	switch s.SecurityMode {
	case "download-scan":
		return "Скачаем во временную папку, проверим и установим."
	case "release-scan":
		return "Подберём GitHub Release, скачиваемый файл будет проверен."
	case "file-scan":
		return "Локальный файл будет проверен перед установкой."
	case "manager-trust":
		return "Установка через системный менеджер и его подписи."
	case "source-build":
		return "Source-build требует доверия к репозиторию; смотри план."
	default:
		return safeLine(s.Detail, 130)
	}
}

func stepForSource(s core.SourcePreview) int {
	if s.UsesPackageTrust {
		return 3
	}
	if s.NeedsDownload {
		return 1
	}
	return 1
}

func (ui *uiState) scan() {
	text := strings.TrimSpace(ui.input.Text)
	if text == "" {
		dialog.ShowInformation("Instally", core.T("add.source"), ui.window)
		return
	}
	ui.lastText = text
	ui.busy(core.T("checking"), 3)
	go func() {
		res := core.ScanInputText(text, ui.securityOptions())
		fyne.Do(func() {
			ui.lastScan = res
			ui.renderScan(res)
			ui.ready()
		})
	}()
}

func (ui *uiState) showPlan() {
	text := strings.TrimSpace(ui.input.Text)
	if text == "" {
		dialog.ShowInformation("Instally", "Добавь источник, чтобы построить план.", ui.window)
		return
	}
	ui.previewPlan()
}

func (ui *uiState) previewPlan() {
	text := strings.TrimSpace(ui.input.Text)
	if text == "" {
		return
	}
	tasks := core.ParseBatchText(text)
	noSecurity := ui.lastScan.OK && ui.lastText == text && ui.lastScan.Safe
	if noSecurity {
		tasks = core.TasksForCheckedInstall(ui.lastScan, tasks)
	}
	plan := core.BuildPlan(tasks, core.Options{Yes: true, DryRun: true, NoSecurity: noSecurity})
	lines := core.PlanLinesForUI(plan, 4)
	summary := core.PlanSummaryForUI(plan)
	if len(plan.Warnings) > 0 {
		summary += "\nПредупреждения: " + safeLine(strings.Join(plan.Warnings, " · "), 220)
	}
	ui.planText.SetText(summary + "\n\n" + strings.Join(shortLines(lines, 180), "\n\n"))
}

func (ui *uiState) safeInstall() {
	text := strings.TrimSpace(ui.input.Text)
	if text == "" {
		dialog.ShowInformation("Instally", core.T("add.source"), ui.window)
		return
	}
	ui.lastText = text
	ui.resultCard.Show()
	ui.resultTitle.SetText("Проверяем")
	ui.resultText.SetText("Проверяем источник. Установка начнётся только если нет опасных признаков.")
	ui.resultBadge.SetText("проверка")
	ui.meta.SetText("")
	ui.checks.Objects = nil
	ui.checks.Refresh()
	ui.busy(core.T("checking"), 3)
	go func() {
		scan := core.ScanInputText(text, ui.securityOptions())
		var buf bytes.Buffer
		buf.WriteString("Instally: проверка источника\n")
		for _, target := range scan.Targets {
			fmt.Fprintf(&buf, "Источник: %s\n", target.Source)
			fmt.Fprintf(&buf, "Итог: %s\n", core.SecurityHumanSummaryForUI(target.Report))
		}
		for _, warn := range scan.Warnings {
			fmt.Fprintf(&buf, "warning: %s\n", warn)
		}
		if !scan.Safe && !ui.allowUnknown.Checked {
			fyne.Do(func() {
				ui.lastScan = scan
				ui.log.SetText(buf.String())
				ui.renderScan(scan)
				ui.status.SetText("Установка остановлена: проверка не дала безопасный результат.")
				ui.ready()
			})
			return
		}
		fyne.Do(func() {
			ui.lastScan = scan
			ui.renderScan(scan)
			ui.busy(core.T("installing.checked"), 4)
		})
		installOpts := ui.options()
		installOpts.NoSecurity = true
		tasks := core.TasksForCheckedInstall(scan, core.ParseBatchText(text))
		plan := core.BuildPlan(tasks, installOpts)
		res := core.RunPlan(plan, false)
		buf.WriteString("\nInstally: установка из проверенного источника\n")
		buf.WriteString(res.Output)
		fyne.Do(func() {
			ui.log.SetText(buf.String())
			ui.resultCard.Show()
			if res.OK {
				ui.setSteps(4)
				ui.resultTitle.SetText("Готово")
				ui.resultText.SetText("Установка выполнена. Подробности в журнале.")
				ui.resultBadge.SetText("готово")
				ui.status.SetText(core.T("ready"))
			} else {
				ui.resultTitle.SetText("Установка остановлена")
				ui.resultText.SetText("Один из шагов не выполнился. Подробности есть в журнале.")
				ui.resultBadge.SetText("ошибка")
				ui.status.SetText("Процесс остановлен")
			}
			ui.ready()
		})
	}()
}

func (ui *uiState) renderScan(res core.ScanInputResult) {
	ui.resultCard.Show()
	if !res.OK || len(res.Targets) == 0 {
		ui.resultTitle.SetText("Не удалось проверить")
		if len(res.Warnings) > 0 {
			ui.resultText.SetText(safeLine(strings.Join(res.Warnings, "\n"), 240))
		} else {
			ui.resultText.SetText("Источник не удалось обработать.")
		}
		ui.resultBadge.SetText("ошибка")
		return
	}
	t := res.Targets[0]
	rep := t.Report
	ui.resultTitle.SetText(core.SecurityHumanTitleForUI(rep))
	ui.resultText.SetText(core.SecurityHumanSummaryForUI(rep))
	ui.resultBadge.SetText(core.SecurityStatusFriendly(rep.Status))
	ui.meta.SetText(fmt.Sprintf("Источник: %s\nФайл: %s · %s", compactUI(t.Source), core.HumanSizeForUI(rep.Size), core.ShortSHAForUI(rep.SHA256)))
	ui.checks.Objects = nil
	for i, c := range rep.Checks {
		if i >= 6 {
			ui.checks.Add(wrappedMuted(fmt.Sprintf("…и ещё %d проверок в журнале", len(rep.Checks)-i)))
			break
		}
		line := widget.NewLabelWithStyle(checkMark(c.Status)+" "+humanCheckName(c.Name), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		line.Wrapping = fyne.TextWrapWord
		detail := wrappedMuted(safeLine(humanCheckDetail(c), 120))
		ui.checks.Add(container.NewVBox(line, detail))
	}
	ui.checks.Refresh()
	ui.log.SetText(core.JSON(res))
	ui.previewPlan()
	if res.Safe || ui.allowUnknown.Checked {
		ui.setSteps(3)
	} else {
		ui.setSteps(2)
	}
}

func humanCheckName(name string) string {
	switch name {
	case "SHA-256":
		return "Хеш файла"
	case "Тип файла":
		return "Тип файла"
	case "ClamAV":
		return "Локальный антивирус"
	case "VirusTotal":
		return "VirusTotal"
	case "Статика":
		return "Быстрый анализ скрипта"
	default:
		return name
	}
}

func humanCheckDetail(c core.SecurityCheck) string {
	if c.Name == "SHA-256" && len(c.Detail) > 34 {
		return core.ShortSHAForUI(c.Detail)
	}
	return c.Detail
}

func compactUI(s string) string {
	return safeMiddle(strings.TrimSpace(s), 58)
}

func safeLine(s string, max int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\r", ""))
	s = strings.Join(strings.Fields(s), " ")
	return safeMiddle(s, max)
}

func safeMiddle(s string, max int) string {
	r := []rune(s)
	if max <= 0 || len(r) <= max {
		return s
	}
	left := max/2 - 1
	right := max - left - 1
	return string(r[:left]) + "…" + string(r[len(r)-right:])
}

func shortLines(lines []string, max int) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, safeLine(line, max))
	}
	return out
}

func (ui *uiState) busy(text string, step int) {
	ui.status.SetText(text)
	ui.setSteps(step)
	ui.progress.Show()
	ui.progress.Start()
	ui.autoBtn.Disable()
	ui.fileBtn.Disable()
}

func (ui *uiState) ready() {
	ui.progress.Stop()
	ui.progress.Hide()
	ui.autoBtn.Enable()
	ui.fileBtn.Enable()
}

func (ui *uiState) setSteps(active int) {
	labels := []string{"Источник", "Загрузка", "Проверка", "Установка"}
	for i, l := range labels {
		icon := "○"
		if active > i+1 {
			icon = "✓"
		} else if active == i+1 {
			icon = "▶"
		}
		ui.steps[i].SetText(icon + " " + l)
	}
}

func checkMark(status string) string {
	switch status {
	case "clean":
		return "✓"
	case "unsafe", "error":
		return "!"
	default:
		return "•"
	}
}
