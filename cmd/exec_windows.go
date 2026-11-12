//go:build windows

package cmd

import (
	"errors"
	"os"
	"os/exec"
)

// execArgv runs argv as a child process and exits with its exit code. Windows
// has no exec(2), so the process cannot be replaced.
func execArgv(argv []string) error {
	c := exec.Command(argv[0], argv[1:]...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := c.Run()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return exitError{code: ee.ExitCode()}
	}
	return err
}
