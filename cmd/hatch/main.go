package main

import (
	"os"

	"github.com/nayeemzen/hatch/internal/hatch"
)

func main() {
	os.Exit(hatch.Main(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
