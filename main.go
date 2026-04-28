package main

import (
	"fmt"
	"os"

	"github.com/nickg/issuefs/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "ifs:", err)
		os.Exit(1)
	}
}
