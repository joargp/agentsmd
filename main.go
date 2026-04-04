package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	check := flag.Bool("check", false, "dry-run: print problems and exit 1 if not correct")
	flag.Parse()

	if err := run(*check); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
