// NewAPI Tools - Docker management platform for newapi
package osutil

import (
	"bufio"
	"os"
	"strings"
)

// parseOSRelease reads and parses /etc/os-release.
func parseOSRelease() (*osRelease, error) {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	release := &osRelease{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := strings.Trim(parts[1], `"'`)

		switch key {
		case "ID":
			release.ID = value
		case "ID_LIKE":
			release.IDLike = value
		case "NAME":
			release.Name = value
		case "VERSION":
			release.Version = value
		case "VERSION_ID":
			release.VersionID = value
		}
	}

	return release, scanner.Err()
}
