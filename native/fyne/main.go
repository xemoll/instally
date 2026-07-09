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
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type uiState struct {
	window       fyne.Window
	app          fyne.App
	input        *widget.Entry
	status       *widget.Label
	sourceIcon   *widget.Icon
	sourceLine   *widget.Label
	steps        []*widget.Label
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
	presetGrid   *fyne.Container
	instBtn      *widget.Button
}

func main() {
	a := fyneapp.NewWithID("dev.instally.native")
	a.Settings().SetTheme(theme.DarkTheme())
	w := a.NewWindow("Instally")
	w.Resize(fyne.NewSize(1000, 750))
	w.SetMaster()

	ui := &uiState{window: w, app: a}
	ui.build(w)
	w.ShowAndRun()
}

func (ui *uiState) build(w fyne.Window) {
	sys := core.Detect()
	installTab := ui.buildInstallTab(sys)
	presetsTab := ui.buildPresetsTab()
	settingsTab := ui.buildSettingsTab()

	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Install", theme.DownloadIcon(), installTab),
		container.NewTabItemWithIcon("Presets", theme.GridIcon(), presetsTab),
		container.NewTabItemWithIcon("Settings", theme.SettingsIcon(), settingsTab),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	w.SetContent(tabs)
	w.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
		if len(uris) == 0 {
			return
		}
		tabs.SelectTabIndex(0)
		ui.input.SetText("local: " + uris[0].Path())
	})
}

func (ui *uiState) buildInstallTab(sys core.SystemInfo) *fyne.Container {
	topBar := container.NewBorder(nil, nil,
		widget.NewLabelWithStyle("Instally", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("● "+core.SystemLabelForUI(sys), fyne.TextAlignTrailing, fyne.TextStyle{Monospace: true}),
	)

	ui.sourceIcon = widget.NewIcon(theme.MailAttachmentIcon())
	ui.sourceLine = widget.NewLabel(core.T("source.hint"))
	sourceRow := container.NewHBox(ui.sourceIcon, ui.sourceLine)

	ui.input = widget.NewEntry()
	ui.input.SetPlaceHolder(core.T("source.placeholder"))
	ui.input.Wrapping = fyne.TextWrapOff
	ui.input.OnChanged = func(s string) {
		ui.lastScan = core.ScanInputResult{}
		ui.inspect(s)
	}
	ui.input.OnSubmitted = func(string) { ui.safeInstall() }

	ui.fileBtn = widget.NewButtonWithIcon(core.T("choose.file"), theme.FileIcon(), func() { ui.chooseFile() })
	ui.fileBtn.Importance = widget.LowImportance
	ui.autoBtn = widget.NewButtonWithIcon(core.T("install.safe"), theme.DownloadIcon(), ui.safeInstall)
	ui.autoBtn.Importance = widget.HighImportance
	btnRow := container.NewHBox(ui.fileBtn, ui.autoBtn)

	ui.status = widget.NewLabel(core.T("ready"))
	ui.progress = widget.NewProgressBarInfinite()
	ui.progress.Hide()
	statusRow := container.NewHBox(ui.status, ui.progress)

	steps := []string{"Source", "Download", "Check", "Install"}
	ui.steps = make([]*widget.Label, 4)
	for i, s := range steps {
		ui.steps[i] = widget.NewLabelWithStyle("○ "+s, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	}
	stepsRow := container.NewGridWithColumns(4, ui.steps[0], ui.steps[1], ui.steps[2], ui.steps[3])

	ui.resultTitle = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	ui.resultText = widget.NewLabel("")
	ui.resultBadge = widget.NewLabel("")
	ui.meta = widget.NewLabel("")
	ui.checks = container.NewVBox()
	resultCard := widget.NewCard("", "", container.NewVBox(ui.resultTitle, ui.resultText, ui.resultBadge, ui.meta, ui.checks))

	ui.planText = widget.NewLabel(core.T("plan.placeholder"))
	scanBtn := widget.NewButton(core.T("scan.only"), ui.scan)
	planBtn := widget.NewButton(core.T("show.plan"), ui.showPlan)
	actionRow := container.NewHBox(scanBtn, planBtn)

	ui.log = widget.NewMultiLineEntry()
	ui.log.SetPlaceHolder(core.T("log.placeholder"))
	ui.log.Wrapping = fyne.TextWrapWord

	stepsCard := widget.NewCard("Progress", "", stepsRow)

	left := container.NewBorder(
		container.NewVBox(topBar, sourceRow, ui.input, btnRow, statusRow, stepsCard, resultCard, actionRow),
		nil, nil, nil,
		ui.planText,
	)
	right := container.NewBorder(
		widget.NewLabelWithStyle("Log", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil, nil, nil,
		ui.log,
	)
	split := container.NewHSplit(left, right)
	split.Offset = 0.5
	return container.NewPadded(split)
}

func (ui *uiState) buildPresetsTab() *fyne.Container {
	presets := core.PresetList()
	grid := container.NewGridWrap(fyne.NewSize(280, 140))
	for _, name := range presets {
		n := name
		apps := strings.Join(core.PresetApps(n), ", ")
		card := widget.NewCard(n, apps,
			widget.NewButtonWithIcon("Install", theme.DownloadIcon(), func() {
				ui.installPreset(n)
			}),
		)
		grid.Add(card)
	}
	scroll := container.NewScroll(grid)
	return container.NewPadded(scroll)
}

func (ui *uiState) buildSettingsTab() *fyne.Container {
	ui.vtKey = widget.NewPasswordEntry()
	ui.vtKey.SetPlaceHolder(core.T("vt.key.placeholder"))
	vtStatus := widget.NewButton("VT Status", func() {
		s := core.VirusTotalStatus()
		dialog.ShowInformation("VirusTotal", core.JSON(s), ui.window)
	})
	vtTest := widget.NewButton("Test Key", func() {
		rep := core.SecuritySelfTest()
		pass := strings.Count(rep, "✓")
		fail := strings.Count(rep, "✗")
		dialog.ShowInformation("Self-Test", fmt.Sprintf("Pass: %d, Fail: %d\n\n%s", pass, fail, rep), ui.window)
	})

	ui.vtUpload = widget.NewCheck(core.T("vt.upload"), nil)
	ui.allowUnknown = widget.NewCheck(core.T("allow.limited"), nil)
	clearVT := widget.NewButton("Clear VT Key", func() {
		core.ClearVirusTotalKey()
		ui.vtKey.SetText("")
	})

	langSelect := widget.NewSelect([]string{"ru", "en", "uk"}, func(v string) {
		core.SetAppLanguage(v)
		_ = core.SaveLanguage(v)
	})
	langSelect.SetSelected(core.AppLanguage())

	themeToggle := widget.NewButton("Toggle Theme", func() {
		if ui.app.Settings().Theme() == theme.DarkTheme() {
			ui.app.Settings().SetTheme(theme.LightTheme())
		} else {
			ui.app.Settings().SetTheme(theme.DarkTheme())
		}
	})

	doctorBtn := widget.NewButton("Run Doctor", func() {
		d := core.Doctor()
		dialog.ShowInformation("Diagnostics", d, ui.window)
	})
	versionBtn := widget.NewButton("Version", func() {
		dialog.ShowInformation("Version", core.VersionInfo()+core.BuildInfo(), ui.window)
	})

	form := container.NewVBox(
		widget.NewCard("Language", "", langSelect),
		widget.NewCard("Appearance", "", themeToggle),
		widget.NewCard("VirusTotal API", "", container.NewVBox(
			ui.vtKey,
			container.NewHBox(vtStatus, vtTest, clearVT),
		)),
		ui.vtUpload,
		ui.allowUnknown,
		widget.NewCard("Diagnostics", "", container.NewHBox(doctorBtn, versionBtn)),
	)
	return container.NewPadded(container.NewVBox(form, layout.NewSpacer()))
}

func (ui *uiState) installPreset(name string) {
	apps := core.PresetApps(name)
	ui.window.SetContent(container.NewBorder(
		widget.NewLabelWithStyle("Installing preset: "+name, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		nil, nil, nil,
		widget.NewLabel("Installing: "+strings.Join(apps, ", ")),
	))
	opts := core.Options{Yes: true}
	tasks := core.TasksForPreset(name)
	plan := core.BuildPlan(tasks, opts)
	res := core.RunPlan(plan, false)
	dialog.ShowInformation("Result", fmt.Sprintf("Preset %s: %t\n\n%s", name, res.OK, res.Output), ui.window)
	ui.window.SetContent(nil)
	ui.build(ui.window)
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
		ui.sourceLine.SetText("Source not recognized.")
		ui.status.SetText("Need to clarify source")
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
		return "Will download to temp, scan and install."
	case "release-scan":
		return "Will pick GitHub Release, file will be scanned."
	case "file-scan":
		return "Local file will be scanned before install."
	case "manager-trust":
		return "Install via system manager with signatures."
	case "source-build":
		return "Source-build needs repo trust; check plan."
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
		dialog.ShowInformation("Instally", "Add a source to build a plan.", ui.window)
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
		summary += "\nWarnings: " + safeLine(strings.Join(plan.Warnings, " · "), 220)
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
	ui.resultTitle.SetText("Checking")
	ui.resultText.SetText("Checking source. Install will start only if safe.")
	ui.resultBadge.SetText("checking")
	ui.meta.SetText("")
	ui.checks.Objects = nil
	ui.checks.Refresh()
	ui.busy(core.T("checking"), 3)
	go func() {
		scan := core.ScanInputText(text, ui.securityOptions())
		var buf bytes.Buffer
		buf.WriteString("Instally: source check\n")
		for _, target := range scan.Targets {
			fmt.Fprintf(&buf, "Source: %s\n", target.Source)
			fmt.Fprintf(&buf, "Result: %s\n", core.SecurityHumanSummaryForUI(target.Report))
		}
		for _, warn := range scan.Warnings {
			fmt.Fprintf(&buf, "warning: %s\n", warn)
		}
		if !scan.Safe && !ui.allowUnknown.Checked {
			fyne.Do(func() {
				ui.lastScan = scan
				ui.log.SetText(buf.String())
				ui.renderScan(scan)
				ui.status.SetText("Install stopped: security check failed.")
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
		buf.WriteString("\nInstally: install from checked source\n")
		buf.WriteString(res.Output)
		fyne.Do(func() {
			ui.log.SetText(buf.String())
			if res.OK {
				ui.setSteps(4)
				ui.resultTitle.SetText("Done")
				ui.resultText.SetText("Install completed. Details in log.")
				ui.resultBadge.SetText("done")
				ui.status.SetText(core.T("ready"))
			} else {
				ui.resultTitle.SetText("Install stopped")
				ui.resultText.SetText("A step failed. See log for details.")
				ui.resultBadge.SetText("error")
				ui.status.SetText("Process stopped")
			}
			ui.ready()
		})
	}()
}

func (ui *uiState) renderScan(res core.ScanInputResult) {
	if !res.OK || len(res.Targets) == 0 {
		ui.resultTitle.SetText("Could not scan")
		if len(res.Warnings) > 0 {
			ui.resultText.SetText(safeLine(strings.Join(res.Warnings, "\n"), 240))
		} else {
			ui.resultText.SetText("Could not process source.")
		}
		ui.resultBadge.SetText("error")
		return
	}
	t := res.Targets[0]
	rep := t.Report
	ui.resultTitle.SetText(core.SecurityHumanTitleForUI(rep))
	ui.resultText.SetText(core.SecurityHumanSummaryForUI(rep))
	ui.resultBadge.SetText(core.SecurityStatusFriendly(rep.Status))
	ui.meta.SetText(fmt.Sprintf("Source: %s\nFile: %s · %s", compactUI(t.Source), core.HumanSizeForUI(rep.Size), core.ShortSHAForUI(rep.SHA256)))
	ui.checks.Objects = nil
	for i, c := range rep.Checks {
		if i >= 6 {
			ui.checks.Add(wrappedMuted(fmt.Sprintf("…and %d more checks in log", len(rep.Checks)-i)))
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

func humanCheckName(name string) string {
	switch name {
	case "SHA-256":
		return "File hash"
	case "File type":
		return "File type"
	case "ClamAV":
		return "Local antivirus"
	case "VirusTotal":
		return "VirusTotal"
	case "Static analysis":
		return "Static script analysis"
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
	labels := []string{"Source", "Download", "Check", "Install"}
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
