// NewAPI Tools - Docker management platform for newapi
package main

import (
	"github.com/simenty/newapi-tools/internal/cli"

	// Trigger init() registration for built-in plugins.
	_ "github.com/simenty/newapi-tools/plugins/newapi"
)

func main() {
	cli.Execute()
}
