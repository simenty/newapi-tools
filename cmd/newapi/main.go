// NewAPI Tools - Docker management platform for newapi
package main

import (
	"github.com/Bonus520/newapi-tools/internal/cli"

	// Trigger init() registration for built-in plugins.
	_ "github.com/Bonus520/newapi-tools/plugins/newapi"
)

func main() {
	cli.Execute()
}
