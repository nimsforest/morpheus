package main

import (
	"github.com/nimsforest/morpheus/internal/cli"
)

// version is set at build time via -ldflags
var version = "dev"

func main() {
	cli.Version = version
	cli.Run()
}
