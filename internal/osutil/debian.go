// NewAPI Tools - Docker management platform for newapi
package osutil

// debianAdapter implements OSAdapter for Debian and Ubuntu.
type debianAdapter struct {
	release *osRelease
}

func (d *debianAdapter) GetPkgManager() string {
	return "apt-get"
}

func (d *debianAdapter) InstallDocker() error {
	// Update package index
	if err := runCmd("apt-get", "update"); err != nil {
		return err
	}
	// Install Docker
	if err := runCmd("apt-get", "install", "-y", "docker.io"); err != nil {
		return err
	}
	// Start Docker service
	return runCmd("systemctl", "start", "docker")
}

func (d *debianAdapter) InstallPackage(name string) error {
	return runCmd("apt-get", "install", "-y", name)
}

func (d *debianAdapter) GetServiceManager() string {
	return "systemctl"
}

func (d *debianAdapter) DetectOS() (string, string, string) {
	return "debian", d.release.ID, d.release.VersionID
}
