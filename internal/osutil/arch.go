// NewAPI Tools - Docker management platform for newapi
package osutil

// archAdapter implements OSAdapter for Arch Linux and derivatives.
type archAdapter struct {
	release *osRelease
}

func (a *archAdapter) GetPkgManager() string {
	return "pacman"
}

func (a *archAdapter) InstallDocker() error {
	// Install Docker via pacman
	if err := runCmd("pacman", "-S", "--noconfirm", "docker"); err != nil {
		return err
	}
	// Start Docker service
	return runCmd("systemctl", "start", "docker")
}

func (a *archAdapter) InstallPackage(name string) error {
	return runCmd("pacman", "-S", "--noconfirm", name)
}

func (a *archAdapter) GetServiceManager() string {
	return "systemctl"
}

func (a *archAdapter) DetectOS() (string, string, string) {
	return "arch", a.release.ID, a.release.VersionID
}
