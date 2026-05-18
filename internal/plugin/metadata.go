// NewAPI Tools - Docker management platform for newapi
package plugin

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// parseYAML is a helper to parse YAML bytes into a target struct.
func parseYAML(data []byte, target interface{}) error {
	return yaml.Unmarshal(data, target)
}

// Metadata represents the plugin metadata parsed from metadata.yml.
type Metadata struct {
	Name        string   `yaml:"name"`
	DisplayName string   `yaml:"display_name"`
	Version     string   `yaml:"version"`
	Description string   `yaml:"description"`
	Commands    []CmdDef `yaml:"commands"`
}

// CmdDef defines a single command in metadata.yml.
type CmdDef struct {
	Name   string `yaml:"name"`
	Script string `yaml:"script"`
	Desc   string `yaml:"desc"`
	Help   string `yaml:"help"`
}

// ParseMetadata reads and parses a metadata.yml file.
func ParseMetadata(dir string) (*Metadata, error) {
	metaPath := filepath.Join(dir, "metadata.yml")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, err
	}

	var meta Metadata
	if err := parseYAML(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}
