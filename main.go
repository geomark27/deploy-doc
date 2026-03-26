package main

import (
	"os"

	"github.com/geomark27/deploy-doc/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}