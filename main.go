// Command release turns a repository's git activity into a version, changelog,
// and tag — with a preview before anything is written.
package main

import (
	"os"

	"github.com/soyagvs/relio/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
