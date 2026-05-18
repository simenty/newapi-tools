// NewAPI Tools - Docker management platform for newapi
package osutil

// fedoraAdapter implements OSAdapter for Fedora.
type fedoraAdapter struct {
	release *osRelease
}

func (f *fedoraAdapter) GetPkgManager() string {
	return "dnf"
}

func (f *fedoraAdapter) InstallDocker() error {
	// Install Docker via dnf
	if err := runCmd("dnf", "install", "-y", "docker"); err != nil {
		return err
	}
	// Start Docker service
	return runCmd("systemctl", "start", "docker")
}

func (f *fedoraAdapter) InstallPackage(name string) error {
	return runCmd("dnf", "install", "-y", name)
}

func (f *fedoraAdapter) GetServiceManager() string {
	return "systemctl"
}

func (f *fedoraAdapter) DetectOS() (string, string, string) {
	return "fedora", f.release.ID, f.release.VersionID
}
