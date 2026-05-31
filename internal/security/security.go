// NewAPI Tools - Security utilities for permission checks and secret masking
package security

import (
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strings"

	"github.com/simenty/newapi-tools/internal/apperr"
)

// MaskSecret masks sensitive strings for safe display.
// Strings shorter than 6 characters are fully masked as "****".
// Strings 6+ characters keep the first 2 and last 2 runes, with "****" in between.
// Uses []rune to correctly handle multi-byte characters (e.g. Chinese, emoji).
// Example: "mysecretpassword" → "my****rd"
func MaskSecret(s string) string {
	r := []rune(s)
	if len(r) < 6 {
		return "****"
	}
	return string(r[:2]) + "****" + string(r[len(r)-2:])
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
		return apperr.Wrap(apperr.CodeConfigLoad, "", err)
	}

	perm := info.Mode().Perm()
	if perm&0077 != 0 {
		return apperr.New(apperr.CodeConfigPerm,
			fmt.Sprintf("config file %s has overly permissive permissions (%04o)", path, perm), "", nil)
	}
	return nil
}

// FixConfigPerm attempts to chmod 600 the given file.
// On Windows it is a no-op. Returns nil if the file doesn't exist.
func FixConfigPerm(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	if err := os.Chmod(path, 0600); err != nil {
		return apperr.New(apperr.CodeConfigPerm, fmt.Sprintf("failed to chmod 600 %s: %v", path, err), "", nil)
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
		return false, apperr.Wrap(apperr.CodeInstallFailed, "", err)
	}

	groupIDs, err := currentUser.GroupIds()
	if err != nil {
		return false, apperr.Wrap(apperr.CodeInstallFailed, "", err)
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
