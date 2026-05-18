// NewAPI Tools - Docker management platform for newapi
package osutil

// rhelAdapter implements OSAdapter for CentOS, Rocky, AlmaLinux, and RHEL.
type rhelAdapter struct {
	release *osRelease
}

func (r *rhelAdapter) GetPkgManager() string {
	return "yum"
}

func (r *rhelAdapter) InstallDocker() error {
	// Install Docker via yum
	if err := runCmd("yum", "install", "-y", "docker"); err != nil {
		return err
	}
	// Start Docker service
	return runCmd("systemctl", "start", "docker")
}

func (r *rhelAdapter) InstallPackage(name string) error {
	return runCmd("yum", "install", "-y", name)
}

func (r *rhelAdapter) GetServiceManager() string {
	return "systemctl"
}

func (r *rhelAdapter) DetectOS() (string, string, string) {
	return "rhel", r.release.ID, r.release.VersionID
}
