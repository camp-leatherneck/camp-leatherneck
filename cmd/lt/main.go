// lt is the Camp Leatherneck CLI for managing multi-agent workspaces.
package main

import (
	"os"

	"github.com/camp-leatherneck/camp-leatherneck/internal/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
