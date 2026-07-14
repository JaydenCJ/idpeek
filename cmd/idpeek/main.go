// Command idpeek decodes UUIDs, ULIDs, KSUIDs, and Snowflake IDs:
// versions, embedded timestamps, and lossless conversions between formats.
package main

import (
	"os"

	"github.com/JaydenCJ/idpeek/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
