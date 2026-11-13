package main

import (
	"fmt"
	"github.com/mrbooshehri/zet-go/internal/cli"
	"os"
)

func main() {
	if err := cli.New(os.Stdout, os.Stderr).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "zet:", err)
		os.Exit(1)
	}
}
