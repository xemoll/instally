package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func LaunchNativeGUI() error {
	name := "instally-native"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	self := SelfPath()
	candidates := []string{}
	if self != "" {
		base := filepath.Dir(self)
		candidates = append(candidates,
			filepath.Join(base, name),
			filepath.Join(base, "dist", runtime.GOOS+"-"+runtime.GOARCH, name),
		)
	}
	if p, err := exec.LookPath(name); err == nil {
		candidates = append(candidates, p)
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			cmd := exec.Command(c)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			return cmd.Run()
		}
	}
	return fmt.Errorf("native GUI binary not found; build it with ./build-native.sh or run: cd native/fyne && go run .")
}
