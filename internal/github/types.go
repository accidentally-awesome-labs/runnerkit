package github

type Repo struct {
	Host     string
	Owner    string
	Name     string
	FullName string
	Private  bool
	Fork     bool
}

type AuthSource struct {
	Kind      string
	Reference string
}

type PermissionStatus struct {
	OK          bool
	Source      AuthSource
	Required    []string
	Remediation []string
}
