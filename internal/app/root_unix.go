//go:build !windows

package app

import "os"

func isRoot() bool { return os.Geteuid() == 0 }
