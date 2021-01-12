package git

type Git struct {
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	SSHKey     string `yaml:"sshKey"`
	Repository string `yaml:"repository"`
	Revision   string `yaml:"revision"`
}
