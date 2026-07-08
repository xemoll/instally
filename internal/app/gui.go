package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type ServerOptions struct {
	Port   int
	NoOpen bool
}

type guiRequest struct {
	Text             string `json:"text"`
	DryRun           bool   `json:"dry_run"`
	Yes              bool   `json:"yes"`
	VirusTotalKey    string `json:"virus_total_key"`
	VirusTotalUpload bool   `json:"virus_total_upload"`
	AllowUnknown     bool   `json:"allow_unknown"`
}

type SourcePreview struct {
	Kind             string `json:"kind"`
	Item             string `json:"item"`
	Title            string `json:"title"`
	Detail           string `json:"detail"`
	SecurityMode     string `json:"security_mode"`
	NeedsDownload    bool   `json:"needs_download"`
	UsesPackageTrust bool   `json:"uses_package_trust"`
}

type InspectResult struct {
	OK       bool            `json:"ok"`
	Sources  []SourcePreview `json:"sources"`
	Warnings []string        `json:"warnings,omitempty"`
}

func InspectInputText(text string) InspectResult {
	tasks := ParseBatchText(text)
	res := InspectResult{OK: len(tasks) > 0}
	if len(tasks) == 0 {
		res.Warnings = append(res.Warnings, "Добавь файл, ссылку, GitHub или имя программы")
		return res
	}
	for _, task := range tasks {
		for _, item := range task.Items {
			res.Sources = append(res.Sources, describeSource(task.Kind, item))
		}
	}
	return res
}

func describeSource(kind, item string) SourcePreview {
	p := SourcePreview{Kind: kind, Item: item, Title: item, Detail: "Источник будет обработан Instally", SecurityMode: "limited"}
	switch kind {
	case "url":
		p.Title = "Ссылка на файл"
		p.Detail = "Instally скачает файл во временную папку, проверит его и только после этого предложит установку"
		p.SecurityMode = "download-scan"
		p.NeedsDownload = true
	case "github", "release":
		p.Title = "GitHub Release"
		p.Detail = "Instally найдёт подходящий файл релиза под твою систему, проверит его и установит из безопасного cache"
		p.SecurityMode = "release-scan"
		p.NeedsDownload = true
	case "local":
		p.Title = "Локальный файл"
		p.Detail = "Instally проверит файл локально, рассчитает хеш и при ключе сверит его через VirusTotal"
		p.SecurityMode = "file-scan"
	case "app", "pkg", "flatpak", "snap", "winget", "scoop", "choco", "brew", "mas":
		p.Title = "Приложение из менеджера"
		p.Detail = "Будет использован системный менеджер пакетов и его проверка подписей"
		p.SecurityMode = "manager-trust"
		p.UsesPackageTrust = true
	case "git":
		p.Title = "Git-репозиторий"
		p.Detail = "Instally скачает исходники, определит способ сборки и покажет план перед установкой"
		p.SecurityMode = "source-build"
		p.NeedsDownload = true
	default:
		p.Title = "Источник"
	}
	return p
}

func (r guiRequest) options() Options {
	sec := SecurityOptionsFromEnv()
	if strings.TrimSpace(r.VirusTotalKey) != "" {
		sec.VirusTotalKey = strings.TrimSpace(r.VirusTotalKey)
	}
	if r.VirusTotalUpload {
		sec.VirusTotalUpload = true
	}
	if r.AllowUnknown {
		sec.AllowUnknown = true
	}
	return Options{Yes: r.Yes, DryRun: r.DryRun, VirusTotalKey: sec.VirusTotalKey, VirusTotalUpload: sec.VirusTotalUpload, AllowUnknown: sec.AllowUnknown}
}

func (r guiRequest) securityOptions() SecurityOptions {
	sec := SecurityOptionsFromEnv()
	if strings.TrimSpace(r.VirusTotalKey) != "" {
		sec.VirusTotalKey = strings.TrimSpace(r.VirusTotalKey)
	}
	if r.VirusTotalUpload {
		sec.VirusTotalUpload = true
	}
	if r.AllowUnknown {
		sec.AllowUnknown = true
	}
	return sec
}

func ServeGUI(opts ServerOptions) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(indexHTML))
	})
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, Detect()) })
	mux.HandleFunc("/api/inspect", func(w http.ResponseWriter, r *http.Request) {
		var req guiRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, InspectInputText(req.Text))
	})
	mux.HandleFunc("/api/doctor", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, map[string]string{"text": Doctor()}) })
	mux.HandleFunc("/api/scan", func(w http.ResponseWriter, r *http.Request) {
		var req guiRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, ScanInputText(req.Text, req.securityOptions()))
	})
	mux.HandleFunc("/api/upload-scan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4<<30)
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		f, header, err := r.FormFile("file")
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		defer f.Close()
		dir := filepath.Join(cacheDir(), "uploads")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		name := sanitizeName(filepath.Base(header.Filename))
		if name == "" || name == "." {
			name = "upload.bin"
		}
		path := filepath.Join(dir, fmt.Sprintf("%d-%s", time.Now().UnixNano(), name))
		out, err := os.Create(path)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		_, copyErr := io.Copy(out, f)
		closeErr := out.Close()
		if copyErr != nil {
			writeJSON(w, map[string]any{"ok": false, "error": copyErr.Error()})
			return
		}
		if closeErr != nil {
			writeJSON(w, map[string]any{"ok": false, "error": closeErr.Error()})
			return
		}
		sec := SecurityOptionsFromEnv()
		sec.VirusTotalKey = strings.TrimSpace(r.FormValue("virus_total_key"))
		sec.VirusTotalUpload = parseBoolString(r.FormValue("virus_total_upload"))
		sec.AllowUnknown = parseBoolString(r.FormValue("allow_unknown"))
		rep := ScanFile(path, sec)
		writeJSON(w, map[string]any{"ok": true, "path": path, "name": header.Filename, "report": rep, "safe": SecurityAllowsInstall(rep, sec.AllowUnknown)})
	})
	mux.HandleFunc("/api/plan", func(w http.ResponseWriter, r *http.Request) {
		var req guiRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, BuildPlan(ParseBatchText(req.Text), req.options()))
	})
	mux.HandleFunc("/api/run", func(w http.ResponseWriter, r *http.Request) {
		var req guiRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		plan := BuildPlan(ParseBatchText(req.Text), req.options())
		writeJSON(w, RunPlan(plan, req.DryRun))
	})
	mux.HandleFunc("/api/run-stream", func(w http.ResponseWriter, r *http.Request) {
		var req guiRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		plan := BuildPlan(ParseBatchText(req.Text), req.options())
		RunPlanStream(plan, req.DryRun, w)
	})
	mux.HandleFunc("/api/safe-run-stream", func(w http.ResponseWriter, r *http.Request) {
		var req guiRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		flush := func() {
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
		fmt.Fprintln(w, "Instally: сначала проверяем источник")
		flush()
		scan := ScanInputText(req.Text, req.securityOptions())
		for _, target := range scan.Targets {
			fmt.Fprintf(w, "\nИсточник: %s\n", target.Source)
			writeSecurityHuman(w, target.Report)
		}
		for _, warn := range scan.Warnings {
			fmt.Fprintf(w, "warning: %s\n", warn)
		}
		flush()
		if !scan.Safe && !req.securityOptions().AllowUnknown {
			fmt.Fprintln(w, "\nInstally: установка остановлена — проверка не завершена или нашла риск")
			return
		}
		fmt.Fprintln(w, "\nInstally: проверка пройдена, устанавливаем из проверенного cache")
		flush()
		installOpts := req.options()
		installOpts.NoSecurity = true
		originalTasks := ParseBatchText(req.Text)
		plan := BuildPlan(TasksForCheckedInstall(scan, originalTasks), installOpts)
		RunPlanStream(plan, req.DryRun, w)
	})
	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(opts.Port))
	if err != nil {
		return err
	}
	url := "http://" + ln.Addr().String()
	fmt.Println("Instally GUI:", url)
	if !opts.NoOpen {
		openBrowser(url)
	}
	return http.Serve(ln, mux)
}

func parseBoolString(v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

func openBrowser(url string) {
	var cmds [][]string
	switch runtime.GOOS {
	case "windows":
		cmds = [][]string{{"rundll32", "url.dll,FileProtocolHandler", url}}
	case "darwin":
		cmds = [][]string{{"open", url}}
	default:
		cmds = [][]string{{"xdg-open", url}, {"gio", "open", url}}
	}
	for _, c := range cmds {
		if commandExists(c[0]) != "" {
			_ = exec.Command(c[0], c[1:]...).Start()
			return
		}
	}
}

const indexHTML = `<!doctype html>
<html lang="ru"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Instally</title>
<style>
:root{color-scheme:light;--bg:#f6f8fc;--card:#fff;--text:#101828;--muted:#667085;--soft:#f8fbff;--line:#e4e7ec;--line2:#edf1f7;--blue:#2563eb;--blue-soft:#eff6ff;--sky:#38bdf8;--sky2:#0ea5e9;--sky-soft:#ecfeff;--green:#079455;--green-soft:#ecfdf3;--amber:#b54708;--amber-soft:#fffaeb;--red:#b42318;--red-soft:#fef3f2;--shadow:0 22px 70px rgba(15,23,42,.08);--r:28px;--mono:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace}*{box-sizing:border-box}html,body{margin:0;min-height:100%}body{background:radial-gradient(circle at top,#fff 0,#f8fafc 45%,#f1f5f9 100%);color:var(--text);font:16px/1.55 Inter,ui-sans-serif,system-ui,-apple-system,Segoe UI,Roboto,Arial,sans-serif}button,input{font:inherit}button{border:0;cursor:pointer}main{width:min(900px,calc(100% - 28px));margin:0 auto;padding:30px 0 56px}.top{display:flex;justify-content:space-between;align-items:center;gap:14px;margin-bottom:14px}.brand{font-size:25px;font-weight:950;letter-spacing:-.04em}.sys{display:flex;align-items:center;gap:8px;border:1px solid var(--line);background:rgba(255,255,255,.94);border-radius:999px;padding:8px 14px;font-size:14px;color:#344054;white-space:nowrap}.dot{width:9px;height:9px;border-radius:50%;background:#22c55e}.shell{background:var(--card);border:1px solid var(--line);border-radius:var(--r);box-shadow:var(--shadow);overflow:hidden}.hero{padding:34px 36px 28px}.hero h1{margin:0 0 10px;font-size:40px;line-height:1.08;letter-spacing:-.065em}.hero p{margin:0 0 24px;max-width:720px;color:var(--muted);font-size:18px}.drop{border:1.5px dashed #bfd2ff;border-radius:24px;background:linear-gradient(180deg,#fcfdff,#f8fbff);padding:32px 38px 28px;text-align:center;transition:.16s ease}.drop.drag{border-color:var(--sky2);background:#f0fbff;transform:translateY(-1px)}.icon{width:64px;height:64px;border-radius:50%;display:grid;place-items:center;margin:0 auto 15px;background:var(--blue-soft);color:var(--blue);font-size:34px;font-weight:950}.dropTitle{font-size:22px;font-weight:950;margin-bottom:4px;letter-spacing:-.025em}.dropText{color:var(--muted);margin-bottom:18px;font-size:16px}.fileBtn{display:inline-flex;align-items:center;justify-content:center;gap:10px;padding:14px 20px;border-radius:14px;background:var(--blue);color:white;font-weight:900;min-width:210px;box-shadow:0 10px 24px rgba(37,99,235,.16)}.or{margin:20px 0 12px;color:#8a94a6;font-size:15px}.inputWrap{position:relative;max-width:720px;margin:0 auto}.source{display:block;width:100%;min-width:0;border:1px solid #d0d5dd;border-radius:14px;background:#fff;padding:15px 18px;color:#344054;outline:none;box-shadow:0 1px 0 rgba(15,23,42,.02);overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.source:focus,.vt:focus{border-color:#7dd3fc;box-shadow:0 0 0 4px rgba(14,165,233,.12)}.sourceHint{display:none;margin-top:14px;border:1px solid #dbeafe;border-radius:18px;background:linear-gradient(180deg,#fff,#f8fbff);padding:16px;text-align:left}.sourceHint.show{display:flex;gap:13px;align-items:flex-start}.hintIcon{width:42px;height:42px;border-radius:15px;background:var(--sky-soft);color:var(--sky2);display:grid;place-items:center;font-weight:950;flex:none}.hintText{min-width:0}.hintTitle{font-weight:950;letter-spacing:-.02em;overflow-wrap:anywhere}.hintDetail{margin-top:3px;color:var(--muted);font-size:14px;overflow-wrap:anywhere}.flow{display:none;margin-top:14px;grid-template-columns:repeat(4,minmax(0,1fr));gap:8px}.flow.show{display:grid}.step{min-width:0;border:1px solid var(--line2);border-radius:14px;padding:10px 8px;text-align:center;font-size:13px;font-weight:950;color:#98a2b3;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.step.done{background:var(--green-soft);border-color:#b9e8cd;color:var(--green)}.step.active{background:var(--sky-soft);border-color:#a5f3fc;color:#0284c7;box-shadow:inset 0 0 0 1px rgba(14,165,233,.08)}.actions{display:grid;grid-template-columns:1fr 1.12fr;gap:16px;margin-top:26px}.primary,.secondary,.install{display:inline-flex;align-items:center;justify-content:center;gap:10px;border-radius:16px;padding:15px 18px;font-weight:950;min-height:58px;text-align:center;line-height:1.2}.secondary{background:#fff;border:1px solid var(--line);color:#111827}.primary,.install{background:linear-gradient(180deg,#7dd3fc,#38bdf8);color:#073142;box-shadow:0 16px 32px rgba(14,165,233,.22)}.primary:hover,.secondary:hover,.install:hover,.fileBtn:hover{filter:brightness(.99);transform:translateY(-1px)}button:disabled{opacity:.55;cursor:not-allowed;transform:none!important;box-shadow:none!important}.result{display:none;border-top:1px solid var(--line);background:#fff;padding:26px 36px 30px}.result.show{display:block}.resultTop{display:flex;justify-content:space-between;align-items:flex-start;gap:16px}.resultText{min-width:0}.result h2{margin:0 0 6px;font-size:26px;line-height:1.15;letter-spacing:-.045em;overflow-wrap:anywhere}.result p{margin:0;color:var(--muted);overflow-wrap:anywhere}.status{flex:none;border-radius:999px;min-width:92px;padding:8px 12px;text-align:center;background:#eef2ff;color:#3657c6;font-size:13px;font-weight:950;white-space:nowrap}.status.clean{background:var(--green-soft);color:var(--green)}.status.limited,.status.warning{background:var(--amber-soft);color:var(--amber)}.status.unsafe,.status.error{background:var(--red-soft);color:var(--red)}.facts{display:grid;grid-template-columns:1fr 1fr;gap:12px;margin-top:18px}.fact{min-width:0;border:1px solid var(--line2);background:#f8fafc;border-radius:16px;padding:14px 16px}.label{font-size:12px;font-weight:900;letter-spacing:.03em;text-transform:uppercase;color:#98a2b3;margin-bottom:6px}.value{font-size:14px;overflow-wrap:anywhere;word-break:break-word}.checks{margin-top:14px}.check{display:flex;gap:12px;padding:12px 0;border-top:1px solid #f1f4f8;min-width:0}.check:first-child{border-top:0}.checkIcon{width:28px;height:28px;border-radius:999px;background:#eef2ff;color:#3657c6;display:grid;place-items:center;font-size:13px;font-weight:950;flex:none}.check.clean .checkIcon{background:var(--green-soft);color:var(--green)}.check.warning .checkIcon,.check.limited .checkIcon,.check.skipped .checkIcon{background:var(--amber-soft);color:var(--amber)}.check.unsafe .checkIcon,.check.error .checkIcon{background:var(--red-soft);color:var(--red)}.checkBody{min-width:0}.checkTitle{font-weight:950;overflow-wrap:anywhere}.checkText{font-size:14px;color:var(--muted);margin-top:2px;overflow-wrap:anywhere}.installRow{display:flex;gap:10px;flex-wrap:wrap;margin-top:18px}.advanced,.logBox{border-top:1px solid var(--line);background:#fbfcfe}.advanced summary,.logBox summary{cursor:pointer;list-style:none;padding:17px 36px;font-weight:950;color:#344054;display:flex;justify-content:space-between;align-items:center;gap:14px}.advanced summary:after,.logBox summary:after{content:'⌄';color:#667085}.advanced summary::-webkit-details-marker,.logBox summary::-webkit-details-marker{display:none}.advBody{border-top:1px solid var(--line2);padding:18px 36px 20px;display:grid;gap:12px}.vt{width:100%;min-width:0;border:1px solid #d0d5dd;border-radius:14px;background:#fff;padding:13px 15px;outline:none}.flags{display:flex;gap:14px;flex-wrap:wrap;color:#475467;font-size:14px}.flags label{display:flex;gap:8px;align-items:center}.log{margin:0 36px 26px;border-top:1px solid var(--line2);padding-top:14px;max-height:270px;overflow:auto;white-space:pre-wrap;font:13px/1.6 var(--mono);color:#111827;overflow-wrap:anywhere}.toast{display:none;position:fixed;left:50%;bottom:22px;transform:translateX(-50%);background:#111827;color:#fff;border-radius:999px;padding:12px 16px;box-shadow:0 18px 42px rgba(0,0,0,.18);font-size:14px;font-weight:900}.toast.show{display:block}.spinner{width:17px;height:17px;border-radius:999px;border:3px solid #e0f2fe;border-top-color:#0284c7;display:inline-block;animation:spin .9s linear infinite;vertical-align:-3px}@keyframes spin{to{transform:rotate(360deg)}}@media(max-width:740px){main{width:min(100% - 18px,900px);padding-top:20px}.top{align-items:flex-start}.brand{font-size:22px}.hero{padding:24px 20px 22px}.hero h1{font-size:30px}.hero p{font-size:16px}.drop{padding:24px 16px}.actions,.facts,.flow{grid-template-columns:1fr}.result{padding:22px 20px 24px}.resultTop{flex-direction:column}.status{align-self:flex-start}.advanced summary,.logBox summary{padding:16px 20px}.advBody{padding:16px 20px 18px}.log{margin:0 20px 22px}.fileBtn,.primary,.secondary,.install{width:100%;min-width:0}.source{white-space:normal}}
</style></head><body><main><div class="top"><div class="brand">Instally</div><div class="sys" id="sys"><span class="dot"></span><span>определяю систему</span></div></div><section class="shell"><div class="hero"><h1>Установить без лишнего</h1><p>Файл, ссылка или GitHub сначала проходят проверку.<br>Установка становится доступной только после результата.</p><div class="drop" id="drop"><div class="icon">↓</div><div class="dropTitle">Перетащи установщик сюда</div><div class="dropText">или выбери файл с компьютера</div><button class="fileBtn" id="choose" type="button">↥ Выбрать файл</button><input id="file" type="file" hidden><div class="or">или вставь ссылку / GitHub / имя программы</div><div class="inputWrap"><input class="source" id="input" spellcheck="false" placeholder="https://example.com/app.AppImage"></div><div class="sourceHint" id="hint"><div class="hintIcon" id="hintIcon">↧</div><div class="hintText"><div class="hintTitle" id="hintTitle">Источник готов</div><div class="hintDetail" id="hintDetail">Instally покажет, что будет сделано перед установкой.</div></div></div><div class="flow" id="flow"><div class="step" id="step1">Источник</div><div class="step" id="step2">Загрузка</div><div class="step" id="step3">Проверка</div><div class="step" id="step4">Установка</div></div></div><div class="actions"><button class="secondary" id="scanBtn" onclick="scan()">♢ Проверить</button><button class="primary" id="autoBtn" onclick="safeInstall()">↓ Проверить и установить</button></div></div><div class="result" id="result"><div class="resultTop"><div class="resultText"><h2 id="title">Готово к проверке</h2><p id="summary">Здесь появится короткий понятный итог.</p></div><div class="status" id="badge">idle</div></div><div class="facts" id="meta"></div><div class="checks" id="checks"></div><div class="installRow"><button class="install" id="installBtn" onclick="install()" disabled>Установить</button><button class="secondary" onclick="dryrun()">Показать план</button></div></div><details class="advanced"><summary>Дополнительно</summary><div class="advBody"><input class="vt" id="vt" type="password" placeholder="VirusTotal API key — необязательно"><div class="flags"><label><input id="upload" type="checkbox">загрузить файл в VirusTotal</label><label><input id="unknown" type="checkbox">разрешить неполную проверку</label></div></div></details><details class="logBox" id="logBox"><summary>Журнал</summary><pre class="log" id="log">Пока пусто.</pre></details></section></main><div class="toast" id="toast"></div>
<script>
const $=id=>document.getElementById(id);let lastSource='',lastAllowed=false,previewTimer=null;function esc(x){return String(x??'').replace(/[&<>]/g,s=>({'&':'&amp;','<':'&lt;','>':'&gt;'}[s]))}function req(){return{text:$('input').value,yes:true,dry_run:false,virus_total_key:$('vt').value,virus_total_upload:$('upload').checked,allow_unknown:$('unknown').checked}}async function get(url){const r=await fetch(url);return await r.json()}async function post(url,obj){const r=await fetch(url,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(obj)});return await r.json()}function toast(t){$('toast').textContent=t;$('toast').className='toast show';setTimeout(()=>$('toast').className='toast',2200)}function progress(active){$('flow').classList.add('show');[1,2,3,4].forEach(i=>{$('step'+i).className='step'+(i<active?' done':i===active?' active':'')})}function hideProgress(){$('flow').classList.remove('show');[1,2,3,4].forEach(i=>$('step'+i).className='step')}function showResult(){$('result').classList.add('show')}function openLog(){$('logBox').open=true;$('log').scrollTop=$('log').scrollHeight}function setBusy(v,msg){document.querySelectorAll('button,input').forEach(x=>x.disabled=v);$('installBtn').disabled=v||!lastAllowed;$('scanBtn').innerHTML=v&&msg?'<span class="spinner"></span>'+msg:'♢ Проверить';$('autoBtn').innerHTML=v&&msg?'<span class="spinner"></span>'+msg:'↓ Проверить и установить'}function statusLabel(s){return {clean:'можно',limited:'неполно',warning:'внимание',unsafe:'опасно',error:'ошибка'}[s]||'готово'}function setStatus(s){$('badge').className='status '+esc(s||'');$('badge').textContent=statusLabel(s)}function human(n){if(!n)return '—';if(n<1024)return n+' B';let u=['KB','MB','GB','TB'],v=n;for(let x of u){v/=1024;if(v<1024)return v.toFixed(1)+' '+x}return n+' B'}function friendlySummary(rep){if(rep.status==='clean')return 'Серьёзных угроз не найдено. Можно продолжить установку.';if(rep.status==='unsafe')return 'Найдены опасные признаки. Установка заблокирована.';if(rep.status==='error')return rep.summary||'Проверка не завершилась.';return 'Проверка неполная: часть способов недоступна. Решение можно принять вручную в «Дополнительно».'}function friendlyTitle(rep){if(rep.status==='clean')return 'Всё выглядит нормально';if(rep.status==='unsafe')return 'Устанавливать нельзя';if(rep.status==='error')return 'Не удалось проверить';return 'Нужна ручная оценка'}function renderMeta(rep,t){let sha=rep.sha256||'—';if(sha.length>30)sha=sha.slice(0,16)+'…'+sha.slice(-10);$('meta').innerHTML='<div class="fact"><div class="label">Источник</div><div class="value">'+esc(t?.source||lastSource||'—')+'</div></div><div class="fact"><div class="label">Файл</div><div class="value">'+esc(human(rep.size))+' · '+esc(sha)+'</div></div>'}function checkIcon(st){return st==='clean'?'✓':(st==='unsafe'||st==='error'?'!':'•')}function renderChecks(rep){let checks=rep.checks||[];if(!checks.length)checks=[{name:'Проверка',status:rep.status||'limited',detail:rep.summary||''}];$('checks').innerHTML=checks.slice(0,6).map(c=>'<div class="check '+esc(c.status||'limited')+'"><div class="checkIcon">'+checkIcon(c.status)+'</div><div class="checkBody"><div class="checkTitle">'+esc(c.name||'Проверка')+'</div><div class="checkText">'+esc(c.detail||'')+'</div></div></div>').join('')}function renderScan(r){showResult();const t=(r.targets||[])[0]||null;const rep=t?.report||r.report||{};if(!t&&!r.report){$('title').textContent='Не удалось проверить';$('summary').textContent=r.error||'Нет результата проверки.';setStatus('error');lastAllowed=false;$('installBtn').disabled=true;return}setStatus(rep.status||'limited');$('title').textContent=rep.title&&rep.title!=='Проверка завершена'?rep.title:friendlyTitle(rep);$('summary').textContent=friendlySummary(rep);renderMeta(rep,t||{source:lastSource});renderChecks(rep);lastAllowed=!!(r.safe||(!rep.blocked&&rep.status==='clean'));$('installBtn').disabled=!lastAllowed}function appendLog(t){$('log').textContent=($('log').textContent==='Пока пусто.'?'':$('log').textContent)+t;$('log').scrollTop=$('log').scrollHeight}async function inspectNow(){let text=$('input').value.trim();if(!text){$('hint').classList.remove('show');hideProgress();return}try{let r=await post('/api/inspect',{text:text});let s=(r.sources||[])[0];if(!s)return;$('hint').classList.add('show');$('hintIcon').textContent=s.needs_download?'↧':(s.uses_package_trust?'◆':'✓');$('hintTitle').textContent=s.title+' · '+s.item;$('hintDetail').textContent=s.detail;progress(s.needs_download?1:3)}catch(e){}}function scheduleInspect(){clearTimeout(previewTimer);previewTimer=setTimeout(inspectNow,220)}async function sourceNeedsDownload(text){try{let r=await post('/api/inspect',{text:text});let s=(r.sources||[])[0];return !!(s&&s.needs_download)}catch(e){return text.startsWith('http')||text.includes('github')||text.startsWith('gh:')}}async function load(){try{let s=await get('/api/status');$('sys').innerHTML='<span class="dot"></span><span>'+esc((s.goos||s.family)+' · '+(s.manager?.id||'none'))+'</span>'}catch(e){$('sys').innerHTML='<span class="dot"></span><span>local</span>'}}async function scan(){lastSource=$('input').value.trim();if(!lastSource){toast('Добавь файл или ссылку');return}setBusy(true,'Проверяю');progress(await sourceNeedsDownload(lastSource)?2:3);$('log').textContent='Проверка: '+lastSource+'
';try{const r=await post('/api/scan',req());progress(3);renderScan(r);appendLog((r.safe?'
Готово: установка доступна.':'
Готово: установка пока недоступна или требует ручного решения.')+'
');toast('Проверка завершена')}catch(e){showResult();$('title').textContent='Ошибка';$('summary').textContent=String(e);setStatus('error');appendLog('
Ошибка: '+e+'
')}setBusy(false)}async function dryrun(){lastSource=$('input').value.trim();if(!lastSource){toast('Добавь файл или ссылку');return}showResult();setBusy(true,'Готовлю');progress(1);$('log').textContent='План без установки: '+lastSource+'
';try{let q=req();q.dry_run=true;const p=await post('/api/plan',q);$('title').textContent='План установки готов';$('summary').textContent='Ничего не установлено. Это только список шагов.';setStatus('limited');$('meta').innerHTML='<div class="fact"><div class="label">Шагов</div><div class="value">'+((p.commands||[]).length)+'</div></div><div class="fact"><div class="label">Менеджер</div><div class="value">'+esc(p.system?.manager?.id||'—')+'</div></div>';$('checks').innerHTML=(p.commands||[]).map(c=>'<div class="check limited"><div class="checkIcon">•</div><div class="checkBody"><div class="checkTitle">'+esc(c.title)+'</div><div class="checkText">'+esc(c.shell||(c.cmd||[]).join(' '))+'</div></div></div>').join('');appendLog((p.warnings||[]).map(w=>'warning: '+w).join('
'));openLog();toast('План готов')}catch(e){appendLog('
Ошибка: '+e+'
');setStatus('error')}setBusy(false)}async function install(){if(!lastAllowed&&!$('unknown').checked){toast('Сначала нужна успешная проверка');return}if(!confirm('Начать установку?'))return;await streamInstall('/api/run-stream')}async function safeInstall(){lastSource=$('input').value.trim();if(!lastSource){toast('Добавь файл или ссылку');return}showResult();await streamInstall('/api/safe-run-stream')}async function streamInstall(endpoint){setBusy(true,'Работаю');progress(await sourceNeedsDownload($('input').value.trim())?2:3);$('log').textContent='Instally: '+$('input').value.trim()+'
';openLog();try{const resp=await fetch(endpoint,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(req())});const reader=resp.body.getReader();const dec=new TextDecoder();let got='';while(true){const {value,done}=await reader.read();if(done)break;let chunk=dec.decode(value,{stream:true});got+=chunk;appendLog(chunk);if(got.includes('провер'))progress(3);if(got.includes('устанавливаем')||got.includes('install'))progress(4)}$('title').textContent=got.includes('остановлена')||got.includes('blocked')?'Установка остановлена':'Готово';$('summary').textContent=got.includes('остановлена')||got.includes('blocked')?'Проверка не дала разрешения на установку. Подробности в журнале.':'Instally закончил выполнение. Подробности доступны в журнале.';setStatus(got.includes('остановлена')||got.includes('blocked')||got.includes('error')?'limited':'clean');toast('Готово')}catch(e){appendLog('
Ошибка: '+e+'
');$('title').textContent='Ошибка установки';$('summary').textContent=String(e);setStatus('error')}setBusy(false)}async function uploadFile(file){if(!file)return;setBusy(true,'Проверяю');progress(3);$('log').textContent='Файл: '+file.name+'
';const fd=new FormData();fd.append('file',file);fd.append('virus_total_key',$('vt').value);fd.append('virus_total_upload',$('upload').checked?'true':'false');fd.append('allow_unknown',$('unknown').checked?'true':'false');try{const r=await fetch('/api/upload-scan',{method:'POST',body:fd}).then(x=>x.json());if(!r.ok){showResult();$('title').textContent='Ошибка загрузки';$('summary').textContent=r.error||'upload failed';setStatus('error');return}$('input').value='local: '+r.path;lastSource=$('input').value;renderScan({safe:r.safe,targets:[{source:'upload: '+r.name,path:r.path,report:r.report}]});appendLog('
Файл загружен во временный cache и проверен.
');toast('Файл проверен')}catch(e){appendLog('
Ошибка: '+e+'
');setStatus('error')}setBusy(false)}function demo(){let params=new URLSearchParams(location.search);if(params.get('demo')==='main'){$('input').value='https://example.com/app.AppImage';inspectNow()}if(params.get('demo')==='result'){ $('input').value='https://example.com/app.AppImage';$('hint').classList.add('show');$('hintIcon').textContent='↧';$('hintTitle').textContent='Ссылка на файл · https://example.com/app.AppImage';$('hintDetail').textContent='Instally скачает файл во временную папку, проверит его и только после этого предложит установку';progress(3); renderScan({safe:true,targets:[{source:'url: app.AppImage',report:{status:'clean',title:'Всё выглядит нормально',summary:'Серьёзных угроз не найдено. Можно продолжить установку.',size:14889779,sha256:'a8f381d2b7b54956b08d91c2',checks:[{name:'Файл опознан',status:'clean',detail:'размер и имя выглядят корректно'},{name:'Хеш сохранён',status:'clean',detail:'файл можно сравнить при повторной загрузке'},{name:'Репутация',status:'clean',detail:'опасных совпадений не найдено'},{name:'Антивирусные движки',status:'clean',detail:'0 / 70 угроз обнаружено'}]}}]}); } }
$('choose').onclick=()=>$('file').click();$('file').onchange=e=>uploadFile(e.target.files[0]);['dragenter','dragover'].forEach(ev=>$('drop').addEventListener(ev,e=>{e.preventDefault();$('drop').classList.add('drag')}));['dragleave','drop'].forEach(ev=>$('drop').addEventListener(ev,e=>{e.preventDefault();$('drop').classList.remove('drag')}));$('drop').addEventListener('drop',e=>uploadFile(e.dataTransfer.files[0]));$('input').addEventListener('input',()=>{lastAllowed=false;$('installBtn').disabled=true;$('result').classList.remove('show');scheduleInspect()});load();demo();
</script></body></html>`

func HTMLForTests() string { return strings.TrimSpace(indexHTML) }
