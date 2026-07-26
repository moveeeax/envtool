package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/moveeeax/envtool/cmd"
)

// exitCoder is implemented by errors that need a specific process exit code
// without main's usual "envtool: <error>" line, because the command has
// already written its own explanation to stdout or stderr.
type exitCoder interface {
	ExitCode() int
}

func main() {
	err := cmd.Execute()
	if err == nil {
		return
	}
	var ec exitCoder
	if errors.As(err, &ec) {
		if msg := err.Error(); msg != "" {
			fmt.Fprintln(os.Stderr, "envtool:", msg)
		}
		os.Exit(ec.ExitCode())
	}
	fmt.Fprintln(os.Stderr, "envtool:", err)
	os.Exit(1)
}
