// Command agent-kits discovers, plans and installs agent capabilities from Git sources.
//
// It never writes to a source and has no publish operation (D-003, D-004).
package main

import (
	"os"

	"github.com/LuchoC-Dev/agent-kits/internal/cli"
)

func main() {
	os.Exit(cli.New().Run(os.Args[1:]))
}
