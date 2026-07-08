package app

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type SecurityOptions struct {
	VirusTotalKey    string `json:"virus_total_key,omitempty"`
	VirusTotalUpload bool   `json:"virus_total_upload"`
	AllowUnknown     bool   `json:"allow_unknown"`
}

type SecurityCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type SecurityReport struct {
	Path     string          `json:"path"`
	Name     string          `json:"name"`
	Size     int64           `json:"size"`
	SHA256   string          `json:"sha256,omitempty"`
	Type     string          `json:"type"`
	Status   string          `json:"status"`
	Title    string          `json:"title"`
	Summary  string          `json:"summary"`
	Blocked  bool            `json:"blocked"`
	Checks   []SecurityCheck `json:"checks"`
	Warnings []string        `json:"warnings,omitempty"`
}

type ScanTarget struct {
	Source string         `json:"source"`
	Kind   string         `json:"kind,omitempty"`
	Item   string         `json:"item,omitempty"`
	Path   string         `json:"path,omitempty"`
	Report SecurityReport `json:"report"`
	Error  string         `json:"error,omitempty"`
}

type ScanInputResult struct {
	OK       bool         `json:"ok"`
	Safe     bool         `json:"safe"`
	Targets  []ScanTarget `json:"targets"`
	Warnings []string     `json:"warnings,omitempty"`
}

func SecurityOptionsFromEnv() SecurityOptions {
	cfg := LoadConfig()
	key := strings.TrimSpace(os.Getenv("INSTALLY_VT_API_KEY"))
	if key == "" {
		key = readKeyFile(strings.TrimSpace(os.Getenv("INSTALLY_VT_KEY_FILE")))
	}
	if key == "" {
		key = strings.TrimSpace(cfg.VirusTotalAPIKey)
	}
	return SecurityOptions{
		VirusTotalKey:    key,
		VirusTotalUpload: boolEnv("INSTALLY_VT_UPLOAD"),
		AllowUnknown:     boolEnv("INSTALLY_ALLOW_UNKNOWN"),
	}
}

func boolEnv(name string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func validateDownloadURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme: %s", u.Scheme)
	}
	if u.Scheme == "http" && !boolEnv("INSTALLY_ALLOW_INSECURE_HTTP") {
		return nil, fmt.Errorf("plain HTTP downloads are blocked by default; use HTTPS or set INSTALLY_ALLOW_INSECURE_HTTP=1 only for a trusted local mirror")
	}
	if u.User != nil {
		return nil, fmt.Errorf("credentials in download URLs are not allowed")
	}
	if u.Hostname() == "" {
		return nil, fmt.Errorf("URL host is empty")
	}
	if !boolEnv("INSTALLY_ALLOW_PRIVATE_URLS") && isPrivateHost(u.Hostname()) {
		return nil, fmt.Errorf("private/local download host is blocked by default: %s; set INSTALLY_ALLOW_PRIVATE_URLS=1 to allow it", u.Hostname())
	}
	return u, nil
}

func isPrivateHost(host string) bool {
	h := strings.ToLower(strings.Trim(host, "[]"))
	if h == "localhost" || strings.HasSuffix(h, ".localhost") || h == "local" {
		return true
	}
	if ip := net.ParseIP(h); ip != nil {
		return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified()
	}
	if boolEnv("INSTALLY_SKIP_DNS_PRIVATE_CHECK") {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, h)
	if err != nil {
		return false
	}
	for _, addr := range addrs {
		ip := addr.IP
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
			return true
		}
	}
	return false
}

func envForSecurity(opts Options) map[string]string {
	env := map[string]string{}
	if opts.VirusTotalKey != "" {
		env["INSTALLY_VT_API_KEY"] = opts.VirusTotalKey
	}
	if opts.VirusTotalUpload {
		env["INSTALLY_VT_UPLOAD"] = "1"
	}
	if opts.AllowUnknown {
		env["INSTALLY_ALLOW_UNKNOWN"] = "1"
	}
	if len(env) == 0 {
		return nil
	}
	return env
}

func envForChildSecurity(opts Options) map[string]string {
	env := map[string]string{}
	// Do not pass VirusTotal API keys directly to child processes.
	// For one-shot --vt-key flows, write an ephemeral 0600 key file and pass
	// only its path. runCommand* removes it after the child exits. Saved keys
	// remain in the normal user config and do not need to be propagated.
	if strings.TrimSpace(opts.VirusTotalKey) != "" && !opts.DryRun {
		if path, err := writeEphemeralKeyFile(opts.VirusTotalKey); err == nil {
			env["INSTALLY_VT_KEY_FILE"] = path
			env["INSTALLY_SECRET_CLEANUP"] = path
		}
	}
	if opts.VirusTotalUpload {
		env["INSTALLY_VT_UPLOAD"] = "1"
	}
	if opts.AllowUnknown {
		env["INSTALLY_ALLOW_UNKNOWN"] = "1"
	}
	if len(env) == 0 {
		return nil
	}
	return env
}

func writeEphemeralKeyFile(key string) (string, error) {
	key = strings.TrimSpace(key)
	if err := validateVirusTotalKey(key); err != nil {
		return "", err
	}
	dir := filepath.Join(dataDir(), "secrets")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	f, err := os.OpenFile(filepath.Join(dir, fmt.Sprintf("vt-key-%d", time.Now().UnixNano())), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString(key); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func ScanFile(path string, opts SecurityOptions) SecurityReport {
	path = expandPath(path)
	rep := SecurityReport{Path: path, Name: filepath.Base(path), Status: "limited", Title: "Проверка ограничена", Summary: "Файл не удалось проверить всеми способами.", Type: "unknown"}
	st, err := os.Stat(path)
	if err != nil {
		rep.Status = "error"
		rep.Title = "Файл не найден"
		rep.Summary = err.Error()
		rep.Blocked = true
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "Файл", Status: "error", Detail: err.Error()})
		return rep
	}
	rep.Size = st.Size()
	if st.IsDir() {
		rep.Type = "directory"
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "Тип", Status: "limited", Detail: "Это папка с исходниками. VirusTotal проверяет отдельные файлы, поэтому онлайн-проверка папки невозможна."})
		runLocalScanner(path, &rep)
		finalizeSecurity(&rep, opts.AllowUnknown)
		return rep
	}
	rep.Type = detectFileType(path)
	runFileStructureCheck(path, &rep)
	runPermissionCheck(path, &rep)
	runFilenamePolicyCheck(path, &rep)
	runArchiveSafetyCheck(path, &rep)
	runInstallerMetadataCheck(path, &rep)
	sha, err := sha256File(path)
	if err != nil {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "SHA-256", Status: "error", Detail: err.Error()})
	} else {
		rep.SHA256 = sha
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "SHA-256", Status: "clean", Detail: sha})
	}
	rep.Checks = append(rep.Checks, SecurityCheck{Name: "Тип файла", Status: "clean", Detail: rep.Type})
	runEmbeddedSignatureCheck(path, &rep)
	runLocalScanner(path, &rep)
	runYARAScanner(path, &rep)
	runPlatformSignatureCheck(path, &rep)
	runStaticHeuristics(path, &rep)
	runVirusTotal(path, opts, &rep)
	finalizeSecurity(&rep, opts.AllowUnknown)
	return rep
}

func SecurityAllowsInstall(rep SecurityReport, allowUnknown bool) bool {
	if rep.Status == "unsafe" || rep.Status == "error" {
		return false
	}
	if rep.Status == "clean" {
		return true
	}
	return allowUnknown
}

func finalizeSecurity(rep *SecurityReport, allowUnknown bool) {
	unsafe := false
	cleanStrong := 0
	limited := false
	warn := false
	for _, c := range rep.Checks {
		switch c.Status {
		case "unsafe":
			unsafe = true
		case "clean":
			if isStrongCleanCheck(c.Name) {
				cleanStrong++
			}
		case "limited", "unknown":
			limited = true
		case "skipped":
			if isRequiredSecurityLayer(c.Name) {
				limited = true
			}
		case "warning":
			warn = true
		}
	}
	if unsafe {
		rep.Status = "unsafe"
		rep.Title = "Опасно: установка заблокирована"
		rep.Summary = "Одна или несколько проверок нашли угрозу. Instally не будет устанавливать этот файл."
		rep.Blocked = true
		return
	}
	if warn {
		rep.Status = "limited"
		rep.Title = "Есть предупреждения"
		rep.Summary = "Явная угроза не найдена, но есть подозрительные признаки или неполная проверка."
		rep.Blocked = !allowUnknown
		return
	}
	if cleanStrong >= 2 && !limited {
		rep.Status = "clean"
		rep.Title = "Серьёзных угроз не найдено"
		rep.Summary = "Доступные проверки не нашли вредоносных признаков. Это не абсолютная гарантия, но файл прошёл включённые уровни проверки."
		rep.Blocked = false
		return
	}
	if cleanStrong >= 2 {
		rep.Status = "limited"
		rep.Title = "Выглядит нормально, но проверка неполная"
		rep.Summary = "Часть обязательных проверок не выполнена или не дала полного результата."
		rep.Blocked = !allowUnknown
		return
	}
	rep.Status = "limited"
	rep.Title = "Проверка ограничена"
	rep.Summary = "Не хватает локального сканера, системной проверки или сильного онлайн-вердикта, чтобы уверенно разрешить установку."
	rep.Blocked = !allowUnknown
}

func isStrongCleanCheck(name string) bool {
	return name == "ClamAV" || name == "Microsoft Defender" || strings.HasPrefix(name, "VirusTotal") || name == "Gatekeeper" || name == "SHA-256" || name == "YARA" || name == "Embedded signatures"
}

func isRequiredSecurityLayer(name string) bool {
	return name == "Локальный антивирус" || name == "Local antivirus" || name == "Microsoft Defender" || name == "ClamAV"
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func detectFileType(path string) string {
	ext := normalizeExt(path)
	f, err := os.Open(path)
	if err != nil {
		return ext
	}
	defer f.Close()
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	mime := http.DetectContentType(buf[:n])
	if ext != "" {
		return ext + " · " + mime
	}
	return mime
}

const eicarTestString = `X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*`

const eicarSHA256 = "275a021bbfb6489e54d471899f7db9d1663fc695ec2fe2a2c4538aabf651fd0f"

func runEmbeddedSignatureCheck(path string, rep *SecurityReport) {
	st, err := os.Stat(path)
	if err != nil || st.IsDir() || st.Size() > 1024*1024 {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "Embedded signatures", Status: "skipped", Detail: "файл не подходит для встроенных быстрых сигнатур"})
		return
	}
	b, err := os.ReadFile(path)
	if err != nil {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "Embedded signatures", Status: "limited", Detail: err.Error()})
		return
	}
	text := string(b)
	if strings.HasPrefix(text, eicarTestString) && len(b) <= 128 {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "Embedded signatures", Status: "unsafe", Detail: "EICAR test signature detected"})
		return
	}
	lower := strings.ToLower(text)
	if strings.Contains(lower, "eicar-standard-antivirus-test-file") {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "Embedded signatures", Status: "warning", Detail: "EICAR-like string found outside strict test-file form"})
		return
	}
	rep.Checks = append(rep.Checks, SecurityCheck{Name: "Embedded signatures", Status: "clean", Detail: "встроенные быстрые сигнатуры не сработали"})
}

func runYARAScanner(path string, rep *SecurityReport) {
	if commandExists("yara") == "" {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "YARA", Status: "skipped", Detail: "yara не найден; это необязательный дополнительный слой"})
		return
	}
	rules := `rule Instally_EICAR_Test { strings: $a = "EICAR-STANDARD-ANTIVIRUS-TEST-FILE" condition: $a }
rule Instally_Suspicious_Shell_Download_Exec { strings: $a = "curl" nocase $b = "wget" nocase $c = "| sh" $d = "| bash" condition: any of them }
rule Instally_Suspicious_PowerShell_Encoded { strings: $a = "powershell" nocase $b = "-enc" nocase $c = "FromBase64String" nocase condition: 2 of them }`
	f, err := os.CreateTemp("", "instally-rules-*.yar")
	if err != nil {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "YARA", Status: "limited", Detail: err.Error()})
		return
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(rules); err != nil {
		_ = f.Close()
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "YARA", Status: "limited", Detail: err.Error()})
		return
	}
	_ = f.Close()
	code, out := runScannerCommand([]string{"yara", "--fail-on-warnings", f.Name(), path})
	if code == 0 && strings.TrimSpace(out) == "" {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "YARA", Status: "clean", Detail: "правила не нашли совпадений"})
		return
	}
	if code == 0 && strings.TrimSpace(out) != "" {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "YARA", Status: "warning", Detail: compact(out, "YARA match")})
		return
	}
	if code == 1 && strings.TrimSpace(out) != "" {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "YARA", Status: "warning", Detail: compact(out, "YARA match")})
		return
	}
	rep.Checks = append(rep.Checks, SecurityCheck{Name: "YARA", Status: "limited", Detail: compact(out, fmt.Sprintf("yara exit code %d", code))})
}

func runLocalScanner(path string, rep *SecurityReport) {
	if commandExists("clamdscan") != "" {
		code, out := runScannerCommand([]string{"clamdscan", "--fdpass", "--infected", path})
		appendClamResult(rep, "ClamAV", code, out)
		return
	}
	if commandExists("clamscan") != "" {
		code, out := runScannerCommand([]string{"clamscan", "--infected", "--no-summary", path})
		appendClamResult(rep, "ClamAV", code, out)
		return
	}
	if runtime.GOOS == "windows" {
		if defender := findDefender(); defender != "" {
			code, out := runScannerCommand([]string{defender, "-Scan", "-ScanType", "3", "-File", path, "-DisableRemediation"})
			if code == 0 {
				rep.Checks = append(rep.Checks, SecurityCheck{Name: "Microsoft Defender", Status: "clean", Detail: compact(out, "scan completed")})
			} else {
				rep.Checks = append(rep.Checks, SecurityCheck{Name: "Microsoft Defender", Status: "warning", Detail: compact(out, fmt.Sprintf("exit code %d", code))})
			}
			return
		}
	}
	rep.Checks = append(rep.Checks, SecurityCheck{Name: "Локальный антивирус", Status: "skipped", Detail: "Не найден clamscan/clamdscan или системный сканер. Установи ClamAV на Linux/macOS или используй Defender на Windows."})
}

func appendClamResult(rep *SecurityReport, name string, code int, out string) {
	if code == 0 {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: name, Status: "clean", Detail: "угрозы не найдены"})
		return
	}
	if code == 1 {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: name, Status: "unsafe", Detail: compact(out, "обнаружена угроза")})
		return
	}
	rep.Checks = append(rep.Checks, SecurityCheck{Name: name, Status: "limited", Detail: compact(out, fmt.Sprintf("сканер завершился с кодом %d", code))})
}

func runScannerCommand(args []string) (int, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err == nil {
		return 0, out.String()
	}
	if ex, ok := err.(*exec.ExitError); ok {
		return ex.ExitCode(), out.String()
	}
	return 127, err.Error() + "\n" + out.String()
}

func findDefender() string {
	if p := commandExists("MpCmdRun.exe"); p != "" {
		return p
	}
	root := os.Getenv("ProgramData")
	if root == "" {
		root = `C:\ProgramData`
	}
	matches, _ := filepath.Glob(filepath.Join(root, "Microsoft", "Windows Defender", "Platform", "*", "MpCmdRun.exe"))
	if len(matches) > 0 {
		return matches[len(matches)-1]
	}
	p := filepath.Join(root, "Microsoft", "Windows Defender", "MpCmdRun.exe")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}

func runPlatformSignatureCheck(path string, rep *SecurityReport) {
	switch runtime.GOOS {
	case "darwin":
		if commandExists("spctl") == "" {
			rep.Checks = append(rep.Checks, SecurityCheck{Name: "Gatekeeper", Status: "skipped", Detail: "spctl не найден"})
			return
		}
		code, out := runScannerCommand([]string{"spctl", "--assess", "--type", "execute", "--verbose=4", path})
		if code == 0 {
			rep.Checks = append(rep.Checks, SecurityCheck{Name: "Gatekeeper", Status: "clean", Detail: compact(out, "подпись/нотаризация принята")})
		} else {
			rep.Checks = append(rep.Checks, SecurityCheck{Name: "Gatekeeper", Status: "warning", Detail: compact(out, "Gatekeeper не принял файл")})
		}
	case "windows":
		if commandExists("powershell") == "" {
			rep.Checks = append(rep.Checks, SecurityCheck{Name: "Подпись Windows", Status: "skipped", Detail: "PowerShell не найден"})
			return
		}
		ps := "(Get-AuthenticodeSignature -LiteralPath " + winPSQuote(path) + ").Status"
		code, out := runScannerCommand([]string{"powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", ps})
		status := strings.TrimSpace(out)
		if code == 0 && status == "Valid" {
			rep.Checks = append(rep.Checks, SecurityCheck{Name: "Подпись Windows", Status: "clean", Detail: "Authenticode: Valid"})
		} else if code == 0 && status != "" {
			rep.Checks = append(rep.Checks, SecurityCheck{Name: "Подпись Windows", Status: "warning", Detail: "Authenticode: " + status})
		} else {
			rep.Checks = append(rep.Checks, SecurityCheck{Name: "Подпись Windows", Status: "limited", Detail: compact(out, "подпись не проверена")})
		}
	default:
		if commandExists("gpg") == "" {
			rep.Checks = append(rep.Checks, SecurityCheck{Name: "Подпись", Status: "skipped", Detail: "На Linux подпись зависит от формата пакета/репозитория. Отдельный .sig не найден и gpg-проверка не выполнялась."})
			return
		}
		sig := path + ".sig"
		if _, err := os.Stat(sig); err == nil {
			code, out := runScannerCommand([]string{"gpg", "--verify", sig, path})
			if code == 0 {
				rep.Checks = append(rep.Checks, SecurityCheck{Name: "Подпись", Status: "clean", Detail: "gpg verify: OK"})
			} else {
				rep.Checks = append(rep.Checks, SecurityCheck{Name: "Подпись", Status: "warning", Detail: compact(out, "gpg verify failed")})
			}
		} else {
			rep.Checks = append(rep.Checks, SecurityCheck{Name: "Подпись", Status: "skipped", Detail: "Рядом с файлом нет .sig"})
		}
	}
}

func runStaticHeuristics(path string, rep *SecurityReport) {
	st, err := os.Stat(path)
	if err != nil || st.IsDir() || st.Size() > 4*1024*1024 {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "Статический анализ", Status: "skipped", Detail: "Файл слишком большой/бинарный для лёгкой эвристики"})
		return
	}
	b, err := os.ReadFile(path)
	if err != nil {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "Статический анализ", Status: "limited", Detail: err.Error()})
		return
	}
	text := strings.ToLower(string(b))
	type rule struct{ pat, label string }
	rules := []rule{
		{"curl ", "curl"}, {"wget ", "wget"}, {"| sh", "pipe-to-sh"}, {"| bash", "pipe-to-bash"},
		{"bash -c", "bash -c"}, {"sh -c", "sh -c"}, {"rm -rf /", "rm -rf /"}, {"mkfs.", "mkfs"},
		{"dd if=", "raw disk write"}, {"chmod 777", "chmod 777"}, {"chmod +s", "setuid"}, {"chattr +i", "immutable file"},
		{"adduser ", "adduser"}, {"useradd ", "useradd"}, {"/etc/sudoers", "sudoers"}, {"crontab", "crontab"},
		{"systemctl enable", "systemd enable"}, {"/etc/systemd/system", "systemd unit"}, {"ld_preload", "LD_PRELOAD"},
		{"ssh-rsa", "embedded ssh key"}, {"id_rsa", "ssh private key name"}, {"begin openssh private key", "private key"},
		{"powershell -enc", "PowerShell encoded"}, {"encodedcommand", "PowerShell EncodedCommand"}, {"frombase64string", "PowerShell base64"},
		{"invoke-expression", "Invoke-Expression"}, {"iex ", "IEX"}, {"set-mppreference -disablerealtimemonitoring", "Defender disable"},
		{"add-mppreference", "Defender exclusion"}, {"certutil -urlcache", "certutil download"}, {"bitsadmin", "bitsadmin download"},
		{"reg add", "registry modification"}, {"currentversion\\run", "startup persistence"}, {"schtasks", "scheduled task"},
		{"launchctl load", "macOS launch agent"}, {"~/library/launchagents", "LaunchAgents"}, {"/library/launchdaemons", "LaunchDaemons"},
	}
	var hits []string
	for _, r := range rules {
		if strings.Contains(text, r.pat) {
			hits = appendUnique(hits, r.label)
		}
	}
	if strings.Contains(text, "curl") && strings.Contains(text, "| bash") {
		hits = appendUnique(hits, "download-and-execute")
	}
	if strings.Contains(text, "wget") && strings.Contains(text, "| sh") {
		hits = appendUnique(hits, "download-and-execute")
	}
	if base64BlobScore(text) >= 1 {
		hits = appendUnique(hits, "large base64-like blob")
	}
	if len(hits) == 0 {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "Статический анализ", Status: "clean", Detail: "опасные шаблоны в скрипте/тексте не найдены"})
		return
	}
	status := "warning"
	if containsAny(hits, "rm -rf /", "raw disk write", "mkfs", "Defender disable") {
		status = "unsafe"
	}
	if containsAny(hits, "download-and-execute") && containsAny(hits, "sudoers", "systemd unit", "startup persistence", "scheduled task", "LaunchAgents", "LaunchDaemons") {
		status = "unsafe"
	}
	rep.Checks = append(rep.Checks, SecurityCheck{Name: "Статический анализ", Status: status, Detail: "найдены потенциально опасные шаблоны: " + strings.Join(hits, ", ")})
}

func base64BlobScore(text string) int {
	run := 0
	best := 0
	for _, r := range text {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '+' || r == '/' || r == '='
		if ok {
			run++
			if run > best {
				best = run
			}
		} else {
			run = 0
		}
	}
	if best >= 500 {
		return 1
	}
	return 0
}

func containsAny(items []string, wants ...string) bool {
	for _, item := range items {
		for _, want := range wants {
			if item == want {
				return true
			}
		}
	}
	return false
}

func runFilenamePolicyCheck(path string, rep *SecurityReport) {
	base := filepath.Base(path)
	lower := strings.ToLower(base)
	if strings.ContainsAny(base, "\x00\n\r\t") {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "Имя файла", Status: "warning", Detail: "имя содержит управляющие символы"})
		return
	}
	for _, r := range base {
		if r == '\u202e' || r == '\u202d' || r == '\u202a' || r == '\u2066' || r == '\u2067' || r == '\u2068' || r == '\u2069' {
			rep.Checks = append(rep.Checks, SecurityCheck{Name: "Имя файла", Status: "warning", Detail: "имя содержит bidi/control символ, возможна маскировка расширения"})
			return
		}
	}
	if strings.Count(lower, ".") >= 2 {
		parts := strings.Split(lower, ".")
		last := "." + parts[len(parts)-1]
		prev := "." + parts[len(parts)-2]
		if isDocumentOrMediaExt(prev) && isExecutableOrInstallerExt(last) {
			rep.Checks = append(rep.Checks, SecurityCheck{Name: "Имя файла", Status: "warning", Detail: "двойное расширение похоже на маскировку: " + compact(base, base)})
			return
		}
	}
	if len(base) > 180 {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "Имя файла", Status: "warning", Detail: "слишком длинное имя файла; показываю сокращённо: " + compact(base, base)})
		return
	}
	rep.Checks = append(rep.Checks, SecurityCheck{Name: "Имя файла", Status: "clean", Detail: "опасных признаков в имени не найдено"})
}

func isExecutableOrInstallerExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".exe", ".msi", ".appx", ".msix", ".appimage", ".run", ".bin", ".sh", ".cmd", ".bat", ".ps1", ".scr", ".com":
		return true
	}
	return false
}

func runInstallerMetadataCheck(path string, rep *SecurityReport) {
	ext := strings.ToLower(normalizeExt(path))
	magic := magicKind(path)
	if isExecutableOrInstallerExt(ext) && (magic == "unknown" || magic == "script") {
		if ext == ".sh" || ext == ".run" || ext == ".bin" || ext == ".ps1" || ext == ".bat" || ext == ".cmd" {
			rep.Checks = append(rep.Checks, SecurityCheck{Name: "Профиль установщика", Status: "limited", Detail: "скриптовый установщик требует статического анализа и доверия к источнику"})
			return
		}
	}
	if (ext == ".deb" || ext == ".rpm" || strings.Contains(ext, ".pkg.tar.")) && rep.Size < 128 {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "Профиль установщика", Status: "warning", Detail: "пакетный файл подозрительно маленький"})
		return
	}
	if ext == ".appimage" && rep.Size < 4096 {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "Профиль установщика", Status: "warning", Detail: "AppImage подозрительно маленький"})
		return
	}
	rep.Checks = append(rep.Checks, SecurityCheck{Name: "Профиль установщика", Status: "clean", Detail: "метаданные/размер не выглядят подозрительно"})
}

func runFileStructureCheck(path string, rep *SecurityReport) {
	st, err := os.Stat(path)
	if err != nil {
		return
	}
	if st.Size() == 0 {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "Структура файла", Status: "warning", Detail: "файл пустой; установщик не должен быть пустым"})
		return
	}
	magic := magicKind(path)
	ext := normalizeExt(path)
	base := strings.ToLower(filepath.Base(path))
	if strings.Contains(base, ".pdf.exe") || strings.Contains(base, ".jpg.exe") || strings.Contains(base, ".png.exe") || strings.Contains(base, ".doc.exe") || strings.Contains(base, ".docx.exe") {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "Структура файла", Status: "warning", Detail: "подозрительное двойное расширение: " + filepath.Base(path)})
		return
	}
	if (magic == "PE" || magic == "ELF" || magic == "Mach-O") && isDocumentOrMediaExt(ext) {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "Структура файла", Status: "warning", Detail: "файл выглядит как исполняемый, но расширение похоже на документ/медиа: " + ext})
		return
	}
	if ext == ".exe" && magic != "PE" && magic != "unknown" {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "Структура файла", Status: "warning", Detail: ".exe не похож на PE-файл: " + magic})
		return
	}
	if ext == ".appimage" && magic != "ELF" {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "Структура файла", Status: "warning", Detail: "AppImage должен быть ELF, обнаружено: " + magic})
		return
	}
	detail := "magic=" + magic
	if ext != "" {
		detail += " ext=" + ext
	}
	rep.Checks = append(rep.Checks, SecurityCheck{Name: "Структура файла", Status: "clean", Detail: detail})
}

func runPermissionCheck(path string, rep *SecurityReport) {
	st, err := os.Lstat(path)
	if err != nil {
		return
	}
	if st.Mode()&os.ModeSymlink != 0 {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "Права и путь", Status: "warning", Detail: "файл является symlink; проверяй реальный путь перед установкой"})
		return
	}
	mode := st.Mode().Perm()
	if mode&0o002 != 0 {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "Права и путь", Status: "warning", Detail: fmt.Sprintf("файл доступен на запись всем пользователям: %04o", mode)})
		return
	}
	rep.Checks = append(rep.Checks, SecurityCheck{Name: "Права и путь", Status: "clean", Detail: fmt.Sprintf("права файла: %04o", mode)})
}

func runArchiveSafetyCheck(path string, rep *SecurityReport) {
	ext := normalizeExt(path)
	switch ext {
	case ".zip":
		runZipSafetyCheck(path, rep)
	case ".tar", ".tar.gz", ".tgz":
		runTarSafetyCheck(path, rep)
	case ".7z", ".rar", ".tar.xz", ".tar.zst", ".tar.bz2":
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "Архив", Status: "limited", Detail: "формат " + ext + " требует внешнего распаковщика; path-traversal будет проверяться перед/после распаковки, если инструмент доступен"})
	default:
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "Архив", Status: "skipped", Detail: "не архивный формат для встроенной проверки"})
	}
}

func runZipSafetyCheck(path string, rep *SecurityReport) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "Архив", Status: "limited", Detail: "zip не прочитан: " + err.Error()})
		return
	}
	defer zr.Close()
	var total uint64
	for i, f := range zr.File {
		if i > 20000 {
			rep.Checks = append(rep.Checks, SecurityCheck{Name: "Архив", Status: "unsafe", Detail: "слишком много файлов в zip; возможна archive-bomb/DoS"})
			return
		}
		if unsafeArchiveName(f.Name) {
			rep.Checks = append(rep.Checks, SecurityCheck{Name: "Архив", Status: "unsafe", Detail: "опасный path-traversal путь внутри zip: " + compact(f.Name, f.Name)})
			return
		}
		if f.FileInfo().Mode()&os.ModeSymlink != 0 {
			rep.Checks = append(rep.Checks, SecurityCheck{Name: "Архив", Status: "unsafe", Detail: "zip содержит symlink/link, установка заблокирована: " + f.Name})
			return
		}
		total += f.UncompressedSize64
		if total > 8*1024*1024*1024 {
			rep.Checks = append(rep.Checks, SecurityCheck{Name: "Архив", Status: "unsafe", Detail: "zip распаковывается больше 8GB; возможна zip-bomb"})
			return
		}
	}
	rep.Checks = append(rep.Checks, SecurityCheck{Name: "Архив", Status: "clean", Detail: fmt.Sprintf("zip: %d файлов, path-traversal не найден", len(zr.File))})
}

func runTarSafetyCheck(path string, rep *SecurityReport) {
	f, err := os.Open(path)
	if err != nil {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "Архив", Status: "limited", Detail: err.Error()})
		return
	}
	defer f.Close()
	var r io.Reader = f
	if strings.HasSuffix(strings.ToLower(path), ".gz") || strings.HasSuffix(strings.ToLower(path), ".tgz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			rep.Checks = append(rep.Checks, SecurityCheck{Name: "Архив", Status: "limited", Detail: "gzip не прочитан: " + err.Error()})
			return
		}
		defer gz.Close()
		r = gz
	}
	tr := tar.NewReader(r)
	count := 0
	var total int64
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			rep.Checks = append(rep.Checks, SecurityCheck{Name: "Архив", Status: "limited", Detail: "tar не прочитан: " + err.Error()})
			return
		}
		count++
		if count > 20000 {
			rep.Checks = append(rep.Checks, SecurityCheck{Name: "Архив", Status: "unsafe", Detail: "слишком много файлов в tar; возможна archive-bomb/DoS"})
			return
		}
		if unsafeArchiveName(h.Name) || unsafeArchiveName(h.Linkname) {
			rep.Checks = append(rep.Checks, SecurityCheck{Name: "Архив", Status: "unsafe", Detail: "опасный path-traversal путь внутри tar: " + h.Name})
			return
		}
		if h.Typeflag == tar.TypeSymlink || h.Typeflag == tar.TypeLink {
			rep.Checks = append(rep.Checks, SecurityCheck{Name: "Архив", Status: "unsafe", Detail: "tar содержит link/symlink, установка заблокирована: " + h.Name})
			return
		}
		total += h.Size
		if total > 8*1024*1024*1024 {
			rep.Checks = append(rep.Checks, SecurityCheck{Name: "Архив", Status: "unsafe", Detail: "tar распаковывается больше 8GB; возможна archive-bomb"})
			return
		}
	}
	rep.Checks = append(rep.Checks, SecurityCheck{Name: "Архив", Status: "clean", Detail: fmt.Sprintf("tar: %d файлов, path-traversal не найден", count)})
}

func unsafeArchiveName(name string) bool {
	name = strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
	if name == "" {
		return false
	}
	if strings.HasPrefix(name, "/") || strings.Contains(name, "\x00") {
		return true
	}
	if len(name) >= 3 && (name[1] == ':' && (name[2] == '/' || name[2] == '\\')) {
		return true
	}
	parts := strings.Split(name, "/")
	for _, p := range parts {
		if p == ".." {
			return true
		}
	}
	return false
}

func magicKind(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return "unknown"
	}
	defer f.Close()
	buf := make([]byte, 8)
	n, _ := f.Read(buf)
	b := buf[:n]
	if len(b) >= 4 && b[0] == 0x7f && b[1] == 'E' && b[2] == 'L' && b[3] == 'F' {
		return "ELF"
	}
	if len(b) >= 2 && b[0] == 'M' && b[1] == 'Z' {
		return "PE"
	}
	if len(b) >= 4 && ((b[0] == 0xfe && b[1] == 0xed && b[2] == 0xfa) || (b[0] == 0xcf && b[1] == 0xfa && b[2] == 0xed) || (b[0] == 0xca && b[1] == 0xfe && b[2] == 0xba && b[3] == 0xbe)) {
		return "Mach-O"
	}
	if len(b) >= 4 && b[0] == 'P' && b[1] == 'K' && b[2] == 3 && b[3] == 4 {
		return "ZIP"
	}
	if len(b) >= 2 && b[0] == 0x1f && b[1] == 0x8b {
		return "GZIP"
	}
	if len(b) >= 8 && string(b[:8]) == "!<arch>\n" {
		return "ar/deb"
	}
	if len(b) >= 4 && b[0] == 0xed && b[1] == 0xab && b[2] == 0xee && b[3] == 0xdb {
		return "RPM"
	}
	if len(b) >= 2 && b[0] == '#' && b[1] == '!' {
		return "script"
	}
	return "unknown"
}

func isDocumentOrMediaExt(ext string) bool {
	switch ext {
	case ".txt", ".pdf", ".jpg", ".jpeg", ".png", ".gif", ".doc", ".docx", ".xls", ".xlsx", ".mp3", ".mp4", ".avi":
		return true
	}
	return false
}

type vtFileResponse struct {
	Data struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			LastAnalysisStats map[string]int `json:"last_analysis_stats"`
		} `json:"attributes"`
	} `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type vtUploadResponse struct {
	Data struct {
		ID string `json:"id"`
	} `json:"data"`
}

type vtAnalysisResponse struct {
	Data struct {
		ID         string `json:"id"`
		Attributes struct {
			Status string         `json:"status"`
			Stats  map[string]int `json:"stats"`
		} `json:"attributes"`
	} `json:"data"`
}

func runVirusTotal(path string, opts SecurityOptions, rep *SecurityReport) {
	key := strings.TrimSpace(opts.VirusTotalKey)
	if key == "" {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "VirusTotal", Status: "skipped", Detail: "API-ключ не указан; VirusTotal пропущен, локальные проверки продолжаются."})
		return
	}
	if rep.SHA256 == "" {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "VirusTotal", Status: "limited", Detail: "нет SHA-256 для lookup"})
		return
	}
	stats, found, detail := vtLookupHash(rep.SHA256, key)
	if found {
		rep.Checks = append(rep.Checks, vtStatsCheck(stats, detail))
		return
	}
	if !opts.VirusTotalUpload {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "VirusTotal", Status: "limited", Detail: detail + "; файл не загружался, потому что upload выключен"})
		return
	}
	rep.Warnings = append(rep.Warnings, "VirusTotal upload включён: отправляемый файл может быть сохранён VirusTotal и доступен security-вендорам; не отправляй приватные данные.")
	maxUpload := vtMaxUploadSize()
	if rep.Size > maxUpload {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "VirusTotal", Status: "limited", Detail: "хеш не найден; файл больше лимита upload " + humanSize(maxUpload) + ": " + humanSize(rep.Size)})
		return
	}
	analysisID, err := vtUploadFile(path, key)
	if err != nil {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "VirusTotal", Status: "limited", Detail: "upload failed: " + err.Error()})
		return
	}
	stats, status, err := vtPollAnalysis(analysisID, key)
	if err != nil {
		rep.Checks = append(rep.Checks, SecurityCheck{Name: "VirusTotal", Status: "limited", Detail: "analysis " + analysisID + ": " + err.Error()})
		return
	}
	rep.Checks = append(rep.Checks, vtStatsCheck(stats, "uploaded analysis "+analysisID+" status="+status))
}

func vtLookupHash(sha, key string) (map[string]int, bool, string) {
	url := vtAPIBase() + "/files/" + sha
	var v vtFileResponse
	status, err := vtJSON("GET", url, key, nil, "", &v)
	if err != nil {
		return nil, false, err.Error()
	}
	if status == 404 {
		return nil, false, "hash not found in VirusTotal"
	}
	if status == 429 {
		return nil, false, "VirusTotal rate limit"
	}
	if status < 200 || status >= 300 {
		msg := strconv.Itoa(status)
		if v.Error != nil {
			msg += " " + v.Error.Message
		}
		return nil, false, "VirusTotal HTTP " + msg
	}
	return v.Data.Attributes.LastAnalysisStats, true, "hash report found"
}

func vtStatsCheck(stats map[string]int, prefix string) SecurityCheck {
	return vtStatsCheckNamed("VirusTotal", stats, prefix)
}

func vtStatsCheckNamed(name string, stats map[string]int, prefix string) SecurityCheck {
	mal := stats["malicious"]
	sus := stats["suspicious"]
	harm := stats["harmless"]
	und := stats["undetected"]
	detail := fmt.Sprintf("%s · malicious=%d suspicious=%d harmless=%d undetected=%d", prefix, mal, sus, harm, und)
	if mal > 0 || sus >= 3 {
		return SecurityCheck{Name: name, Status: "unsafe", Detail: detail}
	}
	if sus > 0 {
		return SecurityCheck{Name: name, Status: "warning", Detail: detail}
	}
	if harm+und > 0 {
		return SecurityCheck{Name: name, Status: "clean", Detail: detail}
	}
	return SecurityCheck{Name: name, Status: "limited", Detail: detail}
}

func runVirusTotalURL(raw string, opts SecurityOptions) SecurityCheck {
	key := strings.TrimSpace(opts.VirusTotalKey)
	if key == "" {
		return SecurityCheck{Name: "VirusTotal URL", Status: "skipped", Detail: "API-ключ не указан; URL reputation пропущена"}
	}
	analysisID, err := vtSubmitURL(raw, key)
	if err != nil {
		return SecurityCheck{Name: "VirusTotal URL", Status: "limited", Detail: err.Error()}
	}
	stats, status, err := vtPollAnalysis(analysisID, key)
	if err != nil {
		return SecurityCheck{Name: "VirusTotal URL", Status: "limited", Detail: "analysis " + analysisID + ": " + err.Error()}
	}
	return vtStatsCheckNamed("VirusTotal URL", stats, "URL analysis "+analysisID+" status="+status)
}

func vtSubmitURL(raw, key string) (string, error) {
	form := strings.NewReader("url=" + url.QueryEscape(raw))
	var v vtUploadResponse
	status, err := vtJSON("POST", vtAPIBase()+"/urls", key, form, "application/x-www-form-urlencoded", &v)
	if err != nil {
		return "", err
	}
	if status < 200 || status >= 300 {
		return "", fmt.Errorf("VirusTotal URL scan HTTP %d", status)
	}
	if v.Data.ID == "" {
		return "", fmt.Errorf("VirusTotal URL scan did not return analysis id")
	}
	return v.Data.ID, nil
}

func vtUploadFile(path, key string) (string, error) {
	st, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	endpoint := vtAPIBase() + "/files"
	if st.Size() > 32*1024*1024 {
		u, err := vtUploadURL(key)
		if err != nil {
			return "", err
		}
		endpoint = u
	}
	return vtUploadFileToEndpoint(path, key, endpoint)
}

type vtUploadURLResponse struct {
	Data  string `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func vtUploadURL(key string) (string, error) {
	var v vtUploadURLResponse
	status, err := vtJSON("GET", vtAPIBase()+"/files/upload_url", key, nil, "", &v)
	if err != nil {
		return "", err
	}
	if status < 200 || status >= 300 {
		msg := fmt.Sprintf("VirusTotal upload_url HTTP %d", status)
		if v.Error != nil && v.Error.Message != "" {
			msg += ": " + v.Error.Message
		}
		return "", fmt.Errorf(msg)
	}
	if strings.TrimSpace(v.Data) == "" {
		return "", fmt.Errorf("VirusTotal upload_url did not return URL")
	}
	return v.Data, nil
}

func vtUploadFileToEndpoint(path, key, endpoint string) (string, error) {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	errc := make(chan error, 1)
	go func() {
		defer close(errc)
		fw, err := mw.CreateFormFile("file", filepath.Base(path))
		if err != nil {
			_ = pw.CloseWithError(err)
			errc <- err
			return
		}
		f, err := os.Open(path)
		if err != nil {
			_ = pw.CloseWithError(err)
			errc <- err
			return
		}
		_, copyErr := io.Copy(fw, f)
		closeErr := f.Close()
		if copyErr != nil {
			_ = pw.CloseWithError(copyErr)
			errc <- copyErr
			return
		}
		if closeErr != nil {
			_ = pw.CloseWithError(closeErr)
			errc <- closeErr
			return
		}
		if err := mw.Close(); err != nil {
			_ = pw.CloseWithError(err)
			errc <- err
			return
		}
		errc <- pw.Close()
	}()
	var v vtUploadResponse
	status, err := vtJSON("POST", endpoint, key, pr, mw.FormDataContentType(), &v)
	if werr := <-errc; err == nil && werr != nil {
		err = werr
	}
	if err != nil {
		return "", err
	}
	if status < 200 || status >= 300 {
		return "", fmt.Errorf("VirusTotal upload HTTP %d", status)
	}
	if v.Data.ID == "" {
		return "", fmt.Errorf("VirusTotal upload did not return analysis id")
	}
	return v.Data.ID, nil
}

func vtMaxUploadSize() int64 {
	limit := int64(650 * 1024 * 1024)
	if v := strings.TrimSpace(os.Getenv("INSTALLY_VT_MAX_UPLOAD_MB")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 && n <= 650 {
			limit = n * 1024 * 1024
		}
	}
	return limit
}

func vtPollAnalysis(id, key string) (map[string]int, string, error) {
	url := vtAPIBase() + "/analyses/" + id
	for i := 0; i < 8; i++ {
		var v vtAnalysisResponse
		status, err := vtJSON("GET", url, key, nil, "", &v)
		if err != nil {
			return nil, "", err
		}
		if status < 200 || status >= 300 {
			return nil, "", fmt.Errorf("VirusTotal analysis HTTP %d", status)
		}
		if v.Data.Attributes.Status == "completed" {
			return v.Data.Attributes.Stats, v.Data.Attributes.Status, nil
		}
		time.Sleep(time.Duration(i+1) * 2 * time.Second)
	}
	return nil, "queued", fmt.Errorf("analysis still queued")
}

func vtAPIBase() string {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("INSTALLY_VT_API_BASE")), "/")
	if base == "" {
		return "https://www.virustotal.com/api/v3"
	}
	return base
}

func vtJSON(method, url, key string, body io.Reader, contentType string, out any) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return 0, err
	}
	req.Header.Set("x-apikey", key)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Instally-Go")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(out); err != nil && resp.StatusCode != 404 {
		return resp.StatusCode, err
	}
	return resp.StatusCode, nil
}

func InstallURLSafe(raw string, opts Options) RunResult {
	var out bytes.Buffer
	if _, err := PreviewURLCachePath(raw); err != nil {
		return RunResult{DryRun: opts.DryRun, OK: false, ExitCode: 2, Output: "URL rejected: " + err.Error() + "\n", Errors: []string{err.Error()}}
	}
	if opts.TrustedOfficialScript && !isTrustedOfficialScriptURL(raw) {
		return RunResult{DryRun: opts.DryRun, OK: false, ExitCode: 2, Output: "trusted official script mode rejected this URL\n", Errors: []string{"trusted official script mode rejected non-allowlisted URL"}}
	}
	if opts.DryRun {
		preview, _ := PreviewURLCachePath(raw)
		fmt.Fprintf(&out, "would download: %s\n", raw)
		fmt.Fprintf(&out, "would save to: %s\n", preview)
		fmt.Fprintf(&out, "would scan cached file before install\n")
		if opts.TrustedOfficialScript {
			fmt.Fprintf(&out, "trusted official script policy: exact allowlisted HTTPS installer URL required\n")
		}
		plan := BuildPlan([]Task{{Kind: "local", Items: []string{preview}}}, Options{Yes: opts.Yes, DryRun: true, NoSecurity: true})
		for i, c := range plan.Commands {
			fmt.Fprintf(&out, "[%d/%d] %s\n%s\n", i+1, len(plan.Commands), c.Title, commandLine(c))
		}
		for _, w := range plan.Warnings {
			fmt.Fprintf(&out, "warning: %s\n", w)
		}
		return RunResult{DryRun: true, OK: true, Output: out.String()}
	}
	secOpts := securityOptionsFromInstallOptions(opts)
	urlCheck := runVirusTotalURL(raw, secOpts)
	if urlCheck.Status == "unsafe" {
		fmt.Fprintf(&out, "VirusTotal URL: %s\n", urlCheck.Detail)
		return RunResult{DryRun: false, OK: false, ExitCode: 2, Output: out.String() + "URL reputation blocked download\n", Errors: []string{"VirusTotal URL reputation blocked download"}}
	}
	if urlCheck.Status != "skipped" {
		fmt.Fprintf(&out, "VirusTotal URL: [%s] %s\n", urlCheck.Status, urlCheck.Detail)
	}
	path, err := DownloadURLToCache(raw)
	if err != nil {
		return RunResult{DryRun: false, OK: false, ExitCode: 1, Output: out.String() + "download failed: " + err.Error() + "\n", Errors: []string{err.Error()}}
	}
	fmt.Fprintf(&out, "downloaded: %s\n", path)
	rep := ScanFile(path, secOpts)
	writeSecurityHuman(&out, rep)
	allowed := SecurityAllowsInstall(rep, opts.AllowUnknown)
	if !allowed && opts.TrustedOfficialScript && isTrustedOfficialScriptURL(raw) && trustedOfficialScriptMayRun(rep) {
		fmt.Fprintf(&out, "Trusted official script policy: exact allowlisted HTTPS installer URL; no unsafe checks found, limited scan accepted.\n")
		allowed = true
	}
	if !allowed {
		return RunResult{DryRun: opts.DryRun, OK: false, ExitCode: 2, Output: out.String(), Errors: []string{"security check blocked installation"}}
	}
	if opts.DryRun {
		fmt.Fprintf(&out, "\nDry-run: установка после проверки не выполнялась. План установки:\n")
		plan := BuildPlan([]Task{{Kind: "local", Items: []string{path}}}, Options{Yes: opts.Yes, DryRun: true, NoSecurity: true})
		for i, c := range plan.Commands {
			fmt.Fprintf(&out, "[%d/%d] %s\n%s\n", i+1, len(plan.Commands), c.Title, commandLine(c))
		}
		for _, w := range plan.Warnings {
			fmt.Fprintf(&out, "warning: %s\n", w)
		}
		return RunResult{DryRun: true, OK: true, Output: out.String()}
	}
	plan := BuildPlan([]Task{{Kind: "local", Items: []string{path}}}, Options{Yes: opts.Yes, DryRun: false, NoSecurity: true})
	r := RunPlan(plan, false)
	out.WriteString("\n--- install ---\n")
	out.WriteString(r.Output)
	r.Output = out.String()
	return r
}

func isTrustedOfficialScriptURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	if u.Scheme != "https" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	path := strings.TrimRight(u.EscapedPath(), "/")
	allowed := false
	switch host {
	case "ollama.com":
		allowed = path == "/install.sh"
	case "claude.ai":
		allowed = path == "/install.sh"
	}
	if !allowed {
		return false
	}
	_, err = validateDownloadURL(raw)
	return err == nil
}

func trustedOfficialScriptMayRun(rep SecurityReport) bool {
	if rep.Status == "unsafe" || rep.Status == "error" {
		return false
	}
	for _, c := range rep.Checks {
		if c.Status == "unsafe" {
			return false
		}
	}
	return true
}

func InstallLocalSafe(path string, opts Options) RunResult {
	rep := ScanFile(path, securityOptionsFromInstallOptions(opts))
	var out bytes.Buffer
	writeSecurityHuman(&out, rep)
	if !SecurityAllowsInstall(rep, opts.AllowUnknown) {
		return RunResult{DryRun: opts.DryRun, OK: false, ExitCode: 2, Output: out.String(), Errors: []string{"security check blocked installation"}}
	}
	if opts.DryRun {
		fmt.Fprintf(&out, "\nDry-run: установка после проверки не выполнялась. План установки:\n")
		plan := BuildPlan([]Task{{Kind: "local", Items: []string{path}}}, Options{Yes: opts.Yes, DryRun: true, NoSecurity: true})
		for i, c := range plan.Commands {
			fmt.Fprintf(&out, "[%d/%d] %s\n%s\n", i+1, len(plan.Commands), c.Title, commandLine(c))
		}
		for _, w := range plan.Warnings {
			fmt.Fprintf(&out, "warning: %s\n", w)
		}
		return RunResult{DryRun: true, OK: true, Output: out.String()}
	}
	plan := BuildPlan([]Task{{Kind: "local", Items: []string{path}}}, Options{Yes: opts.Yes, DryRun: false, NoSecurity: true})
	r := RunPlan(plan, false)
	out.WriteString("\n--- install ---\n")
	out.WriteString(r.Output)
	r.Output = out.String()
	return r
}

func writeSecurityHuman(w io.Writer, rep SecurityReport) {
	fmt.Fprintf(w, "Проверка безопасности: %s\n", rep.Name)
	fmt.Fprintf(w, "Итог: %s — %s\n", rep.Status, rep.Title)
	if rep.Summary != "" {
		fmt.Fprintf(w, "Пояснение: %s\n", rep.Summary)
	}
	if rep.SHA256 != "" {
		fmt.Fprintf(w, "SHA-256: %s\n", rep.SHA256)
	}
	fmt.Fprintf(w, "Размер: %s\nТип: %s\n", humanSize(rep.Size), rep.Type)
	for _, c := range rep.Checks {
		fmt.Fprintf(w, "- [%s] %s: %s\n", c.Status, c.Name, c.Detail)
	}
	for _, warn := range rep.Warnings {
		fmt.Fprintf(w, "warning: %s\n", warn)
	}
}

func ScanInputText(text string, opts SecurityOptions) ScanInputResult {
	res := ScanInputResult{OK: true, Safe: true}
	tasks := ParseBatchText(text)
	if len(tasks) == 0 {
		res.OK = false
		res.Safe = false
		res.Warnings = append(res.Warnings, "Нет файла, ссылки или пакета для проверки")
		return res
	}
	for _, task := range tasks {
		for _, item := range task.Items {
			t := ScanTarget{Source: task.Kind + ": " + item, Kind: task.Kind, Item: item}
			switch task.Kind {
			case "local":
				t.Path = expandPath(item)
				t.Report = ScanFile(t.Path, opts)
			case "url":
				if _, err := PreviewURLCachePath(item); err != nil {
					t.Error = err.Error()
					t.Report = SecurityReport{Path: item, Name: filepath.Base(item), Status: "error", Title: "URL отклонён", Summary: err.Error(), Blocked: true}
					break
				}
				urlCheck := runVirusTotalURL(item, opts)
				if urlCheck.Status == "unsafe" {
					t.Report = SecurityReport{Path: item, Name: filepath.Base(item), Status: "unsafe", Title: "URL заблокирован VirusTotal", Summary: urlCheck.Detail, Blocked: true, Checks: []SecurityCheck{urlCheck}}
					break
				}
				path, err := DownloadURLToCache(item)
				if err != nil {
					t.Error = err.Error()
					t.Report = SecurityReport{Path: item, Name: filepath.Base(item), Status: "error", Title: "Скачивание не удалось", Summary: err.Error(), Blocked: true, Checks: []SecurityCheck{urlCheck}}
				} else {
					t.Path = path
					t.Report = ScanFile(path, opts)
					if urlCheck.Status != "skipped" {
						t.Report.Checks = append([]SecurityCheck{urlCheck}, t.Report.Checks...)
						finalizeSecurity(&t.Report, opts.AllowUnknown)
					}
				}
			case "github", "release":
				path, err := DownloadGitHubReleaseForScan(normalizeGitHubTarget(item))
				if err != nil {
					t.Error = err.Error()
					t.Report = SecurityReport{Path: item, Name: item, Status: "limited", Title: "GitHub Release не скачан", Summary: "Не удалось выбрать готовый release-asset. При установке будет fallback на git/source-build: " + err.Error(), Blocked: !opts.AllowUnknown}
				} else {
					t.Path = path
					t.Report = ScanFile(path, opts)
				}
			default:
				t.Report = scanVirtualSource(task.Kind, item, opts)
			}
			if !SecurityAllowsInstall(t.Report, opts.AllowUnknown) {
				res.Safe = false
			}
			res.Targets = append(res.Targets, t)
		}
	}
	return res
}

func scanVirtualSource(kind, item string, opts SecurityOptions) SecurityReport {
	trusted := map[string]bool{"pkg": true, "flatpak": true, "snap": true, "winget": true, "scoop": true, "choco": true, "brew": true, "mas": true, "app": true}
	if trusted[kind] {
		return SecurityReport{Name: item, Path: item, Status: "clean", Title: "Можно устанавливать через менеджер", Summary: "Отдельного файла до установки нет: проверка будет выполняться средствами системного менеджера, его репозиториями и подписями. Для VirusTotal нужен конкретный файл, поэтому онлайн-скан здесь не применяется.", Blocked: false, Checks: []SecurityCheck{{Name: "Источник", Status: "clean", Detail: "используется системный/официальный менеджер: " + kind}, {Name: "Файл", Status: "skipped", Detail: "готовый файл отсутствует до установки, поэтому SHA/VirusTotal не применяются"}}}
	}
	return SecurityReport{Name: item, Path: item, Status: "limited", Title: "Нужна осторожность", Summary: "Источник устанавливает код или пакет без готового файла для предварительного VirusTotal-скана. Разреши неполную проверку только если доверяешь источнику.", Blocked: !opts.AllowUnknown, Checks: []SecurityCheck{{Name: "Источник", Status: "limited", Detail: "тип источника: " + kind}, {Name: "Совет", Status: "limited", Detail: "лучше использовать GitHub release, AppImage/deb/rpm/pkg или официальный пакетный менеджер"}}}
}

func PreviewURLCachePath(raw string) (string, error) {
	u, err := validateDownloadURL(raw)
	if err != nil {
		return "", err
	}
	name := filepath.Base(u.Path)
	if inferred := inferFilenameFromURL(u); inferred != "" {
		name = inferred
	}
	if name == "." || name == "/" || name == "" {
		name = "download.bin"
	}
	name = sanitizeName(name)
	if name == "" || name == "." {
		name = "download.bin"
	}
	return filepath.Join(cacheDir(), "downloads", sanitizeName(u.Host), name), nil
}

func inferFilenameFromURL(u *url.URL) string {
	host := strings.ToLower(u.Hostname())
	path := strings.ToLower(strings.TrimRight(u.EscapedPath(), "/"))
	format := strings.ToLower(strings.TrimSpace(u.Query().Get("format")))
	if host == "discord.com" && path == "/api/download" {
		switch format {
		case "deb":
			return "discord.deb"
		case "rpm":
			return "discord.rpm"
		case "tar.gz", "tar":
			return "discord.tar.gz"
		case "pkg.tar.zst":
			return "discord.pkg.tar.zst"
		}
	}
	return ""
}

func DownloadURLToCache(raw string) (string, error) {
	preview, err := PreviewURLCachePath(raw)
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(preview)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, fmt.Sprintf("%d-%s", time.Now().UnixNano(), filepath.Base(preview)))
	return path, downloadFile(raw, path)
}

func DownloadGitHubReleaseForScan(ownerRepo string) (string, error) {
	ownerRepo = normalizeGitHubTarget(ownerRepo)
	asset, err := latestGitHubAsset(ownerRepo, Detect())
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cacheDir(), "downloads", sanitizeName(ownerRepo))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, asset.Name)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	return path, downloadFile(asset.URL, path)
}

func TasksForCheckedInstall(scan ScanInputResult, original []Task) []Task {
	var out []Task
	for _, target := range scan.Targets {
		if target.Kind == "" || target.Item == "" {
			continue
		}
		if target.Path != "" && (target.Kind == "local" || target.Kind == "url" || target.Kind == "github" || target.Kind == "release") {
			out = append(out, Task{Kind: "local", Items: []string{target.Path}})
			continue
		}
		out = append(out, Task{Kind: target.Kind, Items: []string{target.Item}})
	}
	if len(out) == 0 {
		return mergeTasks(original)
	}
	return mergeTasks(out)
}

func humanSize(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	units := []string{"KB", "MB", "GB", "TB"}
	v := float64(n)
	for _, u := range units {
		v /= 1024
		if v < 1024 {
			return fmt.Sprintf("%.1f %s", v, u)
		}
	}
	return fmt.Sprintf("%d B", n)
}

func compact(text, fallback string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return fallback
	}
	text = strings.ReplaceAll(text, "\r", "")
	lines := strings.Split(text, "\n")
	var out []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			out = append(out, l)
		}
		if len(out) >= 3 {
			break
		}
	}
	return strings.Join(out, " · ")
}

func SecuritySelfTest() string {
	f, err := os.CreateTemp("", "instally-eicar-*.com")
	if err != nil {
		return "security self-test failed: " + err.Error() + "\n"
	}
	defer os.Remove(f.Name())
	_, _ = f.WriteString(eicarTestString)
	_ = f.Close()
	rep := ScanFile(f.Name(), SecurityOptions{AllowUnknown: false})
	var b strings.Builder
	fmt.Fprintf(&b, "Instally security self-test\n")
	fmt.Fprintf(&b, "EICAR test file: %s\n", f.Name())
	fmt.Fprintf(&b, "Result: %s — %s\n", rep.Status, rep.Title)
	for _, c := range rep.Checks {
		fmt.Fprintf(&b, "- %s: %s — %s\n", c.Name, c.Status, compact(c.Detail, c.Detail))
	}
	if rep.Status == "unsafe" {
		fmt.Fprintf(&b, "OK: test signature was detected and installation would be blocked.\n")
	} else {
		fmt.Fprintf(&b, "WARNING: test signature was not blocked. Check local scanner/YARA settings.\n")
	}
	return b.String()
}
