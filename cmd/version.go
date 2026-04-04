package cmd

import "fmt"

// Version is set at build time via:
//
//	-ldflags "-X github.com/geomark27/deploy-doc/cmd.Version=v1.1.0"
//
// Falls back to "dev" when running via go run or make run.
var Version = "dev"

func runVersion(_ []string) error {
	fmt.Printf("deploy-doc %s\n", Version)
	return nil
}
