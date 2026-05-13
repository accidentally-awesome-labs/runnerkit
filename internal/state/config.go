package state

import "path/filepath"

const ProjectConfigRelativePath = ".runnerkit/config.yaml"

type ProjectConfig struct {
	Defaults ProjectDefaults `yaml:"defaults" json:"defaults"`
}

type ProjectDefaults struct {
	Repo          string   `yaml:"repo,omitempty" json:"repo,omitempty"`
	Mode          string   `yaml:"mode,omitempty" json:"mode,omitempty"`
	OS            string   `yaml:"os,omitempty" json:"os,omitempty"`
	Arch          string   `yaml:"arch,omitempty" json:"arch,omitempty"`
	ExtraPackages []string `yaml:"extra_packages,omitempty" json:"extra_packages,omitempty"`
}

func ProjectConfigPath(projectRoot string) string {
	return filepath.Join(projectRoot, ProjectConfigRelativePath)
}
