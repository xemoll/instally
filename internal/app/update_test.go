package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.2.0", "1.2.0", 0},
		{"1.2.0", "1.3.0", -1},
		{"1.3.0", "1.2.0", 1},
		{"v1.2.0", "1.2.0", 0},
		{"1.2", "1.2.0", 0},
		{"2.0.0", "1.9.9", 1},
		{"1.2.0", "1.2.1", -1},
		{"1.2.0", "v1.2.0", 0},
		{"1.2.0", "1.2", 0},
		{"2.0", "1.9.9", 1},
		{"0.9", "1.0", -1},
		{"v2.0.0", "v1.9.9", 1},
		{"1.0.0", "1.0.0", 0},
		{"1.0.1", "1.0.0", 1},
	}
	for _, tc := range tests {
		t.Run(tc.a+"_"+tc.b, func(t *testing.T) {
			got := CompareVersions(tc.a, tc.b)
			if got != tc.want {
				t.Fatalf("CompareVersions(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestSelfUpdateCheck_returnsStruct(t *testing.T) {
	info := SelfUpdateCheck()
	if info.Current != appVersion {
		t.Fatalf("Current = %q, want %q", info.Current, appVersion)
	}
	if info.Available {
		t.Log("update available (requires network)")
	}
}

func TestSelfUpdateCheck_fieldsPresent(t *testing.T) {
	info := SelfUpdateCheck()
	if info.Current == "" {
		t.Fatal("Current must not be empty")
	}
	if info.Error != "" {
		t.Logf("network error (expected in offline tests): %s", info.Error)
	}
	if info.Available && info.AssetURL == "" {
		t.Fatal("Available=true but AssetURL is empty")
	}
}

func TestSelfUpdate_dryRun(t *testing.T) {
	info := UpdateInfo{
		Available: true,
		Current:   "1.0.0",
		Latest:    "2.0.0",
		AssetName: "instally-linux-amd64.tar.gz",
		AssetURL:  "https://github.com/xemoll/instally/releases/download/v2.0.0/instally-linux-amd64.tar.gz",
		Size:      4_194_304,
	}
	res := SelfUpdate(Options{DryRun: true}, info)
	if !res.OK {
		t.Fatalf("dry-run should succeed, got errors: %v", res.Errors)
	}
	if !res.DryRun {
		t.Fatal("res.DryRun should be true")
	}
	out := res.Output
	if !strings.Contains(out, "dry-run") {
		t.Fatalf("output should mention dry-run: %s", out)
	}
	if !strings.Contains(out, "1.0.0") || !strings.Contains(out, "2.0.0") {
		t.Fatalf("output should show version transition: %s", out)
	}
	if !strings.Contains(out, "instally-linux-amd64.tar.gz") {
		t.Fatalf("output should contain asset name: %s", out)
	}
	if !strings.Contains(out, "would download") || !strings.Contains(out, "would save to") || !strings.Contains(out, "would replace") {
		t.Fatalf("dry-run output missing expected 'would' actions: %s", out)
	}
}

func TestSelfUpdate_dryRunAlreadyUpToDate(t *testing.T) {
	info := UpdateInfo{
		Available: false,
		Current:   "1.2.0",
		Latest:    "1.2.0",
	}
	res := SelfUpdate(Options{DryRun: true}, info)
	if !res.OK {
		t.Fatalf("up-to-date dry-run should succeed, got: %v", res.Errors)
	}
	out := res.Output
	if !strings.Contains(out, "Already up to date") {
		t.Fatalf("output should show already up-to-date: %s", out)
	}
}

func TestSelfUpdate_dryRunWithErrorInfo(t *testing.T) {
	info := UpdateInfo{
		Available: false,
		Current:   "1.0.0",
		Error:     "simulated error",
	}
	res := SelfUpdate(Options{DryRun: true}, info)
	if res.OK {
		t.Fatal("dry-run with error info should have OK=false")
	}
	out := res.Output
	if !strings.Contains(out, "error:") || !strings.Contains(out, "simulated error") {
		t.Fatalf("output should contain error message: %s", out)
	}
}

func TestCopyFile(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	src := filepath.Join(srcDir, "source.bin")
	dst := filepath.Join(dstDir, "destination.bin")

	content := []byte("mock binary content for update test")
	if err := os.WriteFile(src, content, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("reading dest: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("content mismatch: got %q, want %q", string(got), string(content))
	}

	fi, err := os.Stat(dst)
	if err != nil {
		t.Fatalf("stat dest: %v", err)
	}
	if fi.Mode()&0o111 == 0 {
		t.Fatal("destination should be executable (0o755)")
	}
	_ = os.Remove(dst)
}

func TestCopyFile_nonexistentSource(t *testing.T) {
	err := copyFile("/nonexistent/path/file.bin", filepath.Join(t.TempDir(), "out.bin"))
	if err == nil {
		t.Fatal("expected error for nonexistent source")
	}
}

func TestCopyFile_createsIntermediateDirs(t *testing.T) {
	base := t.TempDir()
	src := filepath.Join(base, "src.bin")
	dst := filepath.Join(base, "a", "b", "c", "dst.bin")
	content := []byte("nested copy test")
	if err := os.WriteFile(src, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile to nested path failed: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("reading nested dest: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("nested content mismatch: got %q, want %q", string(got), string(content))
	}
}

func TestCopyFile_usesTempFileAndRenames(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "original.bin")
	dst := filepath.Join(dir, "final.bin")
	if err := os.WriteFile(src, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatalf("temp file should be cleaned up, found: %s", e.Name())
		}
	}
}

func TestCopyFile_verifyNoSrcRemoved(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.bin")
	dst := filepath.Join(dir, "dst.bin")
	content := []byte("source must survive copy")
	if err := os.WriteFile(src, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(src); os.IsNotExist(err) {
		t.Fatal("source file was removed after copy")
	}
}

func TestCompareVersions_edgeCases(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"1.0.0", "", 1},
		{"", "1.0.0", -1},
		{"v1.0.0-beta", "v1.0.0", -1},
		{"1.0.0-alpha", "1.0.0-beta", -1},
		{"v1.0.0", "v1.0.0-rc1", 1},
		{"v1.0.0+build1", "v1.0.0+build2", 0},
		{"1.0.0+build1", "1.0.0+build2", 0},
		{"v2.0.0+build.1", "v2.0.0+build.2", 0},
		{"1.0.0-alpha.1", "1.0.0-alpha.2", -1},
		{"1.0.0-rc.1", "1.0.0", -1},
	}
	for _, tc := range tests {
		t.Run(tc.a+"_"+tc.b, func(t *testing.T) {
			got := CompareVersions(tc.a, tc.b)
			if got != tc.want {
				t.Fatalf("CompareVersions(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestCompareVersions_invalidInputs(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"abc", "1.0.0", -1},
		{"1.0.0", "abc", 1},
		{"abc", "def", 0},
		{"notaversion", "alsonot", 0},
		{"1.2.3.4", "1.2.3", 1},
		{"1.2", "1.2.3.4", -1},
	}
	for _, tc := range tests {
		t.Run(tc.a+"_"+tc.b, func(t *testing.T) {
			got := CompareVersions(tc.a, tc.b)
			if got != tc.want {
				t.Fatalf("CompareVersions(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestCopyFile_readError(t *testing.T) {
	err := copyFile("/dev/null/nonexistent", filepath.Join(t.TempDir(), "out"))
	if err == nil {
		t.Fatal("expected error for unreadable source")
	}
}

func TestCopyFile_writeError(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.bin")
	if err := os.WriteFile(src, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	// try to write to a directory (should fail)
	err := copyFile(src, dir)
	if err == nil {
		t.Fatal("expected error when destination is a directory")
	}
}
