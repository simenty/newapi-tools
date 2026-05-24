// NewAPI Tools - Docker management platform for newapi
// Package newapi provides the newapi plugin implementation.
//
// This package is reserved for a future GoPlugin implementation.
// Currently, the newapi plugin runs as a ShellPlugin via metadata.yml + scripts/.
// When migrating from Plan B (Shell plugins) to Plan A (Go plugins),
// implement the plugin.Plugin interface here and add a plugin.go file
// to this directory. The Loader will detect the Go binary and prefer it
// over the shell scripts.
package newapi
