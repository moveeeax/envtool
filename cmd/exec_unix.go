//go:build !windows

package cmd

import (
	"os"
	"os/exec"
	"syscall"
)

// execArgv replaces the envtool process with argv, so the program becomes the
// container's main process and receives its signals directly. It only returns
// on failure.
func execArgv(argv []string) error {
	path, err := exec.LookPath(argv[0])
	if err != nil {
		return err
	}
	return syscall.Exec(path, argv, os.Environ())
}
