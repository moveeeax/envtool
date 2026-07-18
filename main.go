package main

import (
	"fmt"
	"os"

	"github.com/cybercapybara/envtool/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "envtool:", err)
		os.Exit(1)
	}
}
