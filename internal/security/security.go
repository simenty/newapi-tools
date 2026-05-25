// NewAPI Tools - Security utilities for permission checks and secret masking
package security

import (
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strings"
)

// MaskSecret masks sensitive strings for safe display.
// Strings shorter than 6 characters are fully masked as "****".
// Strings 6+ characters keep the first 2 and last 2 characters, with "****" in between.
// Example: "mysecretpassword" → "my****rd"
func MaskSecret(s string) string {
	if len(s) < 6 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}

// CheckConfigPerm checks whether a configuration file has overly permissive access.
// On Linux, if the file is readable/writable by group or other (perm & 0077 != 0),
// it returns an error. On Windows, it always returns nil (permissions are handled differently).
func CheckConfigPerm(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, no permission issue
		}
		return fmt.Errorf("failed to check config permissions: %w", err)
	}

	perm := info.Mode().Perm()
	if perm&0077 != 0 {
		return fmt.Errorf("config file %s has overly permissive permissions (%04o); recommend chmod 600", path, perm)
	}
	return nil
}

// CheckDockerGroup checks whether the current user is a member of the docker group.
// On Linux, it checks the user's group memberships. On Windows, it returns (true, nil)
// since Docker Desktop handles group membership differently.
func CheckDockerGroup() (bool, error) {
	if runtime.GOOS == "windows" {
		return true, nil
	}

	currentUser, err := user.Current()
	if err != nil {
		return false, fmt.Errorf("failed to get current user: %w", err)
	}

	groupIDs, err := currentUser.GroupIds()
	if err != nil {
		return false, fmt.Errorf("failed to get user groups: %w", err)
	}

	for _, gid := range groupIDs {
		group, err := user.LookupGroupId(gid)
		if err != nil {
			continue // Skip groups we can't look up
		}
		if strings.EqualFold(group.Name, "docker") {
			return true, nil
		}
	}

	return false, nil
}
