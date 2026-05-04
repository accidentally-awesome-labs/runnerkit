package state

import (
	"errors"
	"time"

	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
)

const SchemaVersion = "2"

var ErrRepositoryExists = errors.New("repository state already exists")

// State is RunnerKit's user-local, versioned, secret-free inventory.
type State struct {
	SchemaVersion string            `json:"schema_version"`
	Repositories  []RepositoryState `json:"repositories"`
}

type RepositoryState struct {
	Repo                   gh.Repo               `json:"repo"`
	Auth                   AuthReference         `json:"auth"`
	Runner                 RunnerIdentity        `json:"runner"`
	Machine                MachineRef            `json:"machine"`
	Provider               ProviderRef           `json:"provider"`
	Cleanup                CleanupMetadata       `json:"cleanup"`
	Safety                 SafetyMetadata        `json:"safety"`
	Ephemeral              EphemeralMetadata     `json:"ephemeral,omitempty"`
	Operations             []OperationCheckpoint `json:"operations,omitempty"`
	RunnerKitVersion       string                `json:"runnerkit_version"`
	RunnerTemplateVersion  string                `json:"runner_template_version,omitempty"`
	ServiceTemplateVersion string                `json:"service_template_version,omitempty"`
	CreatedAt              time.Time             `json:"created_at"`
	UpdatedAt              time.Time             `json:"updated_at"`
}

// EphemeralMetadata records ephemeral runner lifecycle facts that
// `status`, `logs`, `doctor`, `down`, and `destroy` all consume. The
// fields are persisted in user-local state with the `ephemeral` key
// so older states without the field continue to load.
type EphemeralMetadata struct {
	Enabled         bool       `json:"enabled"`
	TTL             string     `json:"ttl,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	LogArchivePath  string     `json:"log_archive_path,omitempty"`
	FinalizerStatus string     `json:"finalizer_status,omitempty"`
	CleanupCommand  string     `json:"cleanup_command,omitempty"`
}

// AuthReference records where auth came from, never the credential value.
type AuthReference struct {
	Source    string `json:"source"`
	Reference string `json:"reference"`
}

type RunnerIdentity struct {
	Name            string   `json:"name"`
	Labels          []string `json:"labels"`
	WorkflowSnippet string   `json:"workflow_snippet,omitempty"`
	Mode            string   `json:"mode"`
	OS              string   `json:"os"`
	Arch            string   `json:"arch"`
}

type MachineRef struct {
	Kind               string     `json:"kind"`
	HostRef            string     `json:"host_ref,omitempty"`
	User               string     `json:"user,omitempty"`
	Port               int        `json:"port,omitempty"`
	KeyPathRef         string     `json:"key_path_ref,omitempty"`
	HostKeyAlgorithm   string     `json:"host_key_algorithm,omitempty"`
	HostKeyFingerprint string     `json:"host_key_fingerprint,omitempty"`
	HostKeyAcceptedAt  *time.Time `json:"host_key_accepted_at,omitempty"`
	InstallPath        string     `json:"install_path,omitempty"`
	WorkDir            string     `json:"work_dir,omitempty"`
	ServiceName        string     `json:"service_name,omitempty"`
}

type ProviderRef struct {
	Kind        string            `json:"kind"`
	Name        string            `json:"name,omitempty"`
	IDs         map[string]string `json:"ids,omitempty"`
	Region      string            `json:"region,omitempty"`
	Profile     string            `json:"profile,omitempty"`
	ResourceIDs map[string]string `json:"resource_ids,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	Cloud       CloudInventory    `json:"cloud,omitempty"`
}

type CloudInventory struct {
	Provider          string            `json:"provider"`
	ServerID          string            `json:"server_id,omitempty"`
	ServerName        string            `json:"server_name,omitempty"`
	ServerStatus      string            `json:"server_status,omitempty"`
	Region            string            `json:"region,omitempty"`
	Datacenter        string            `json:"datacenter,omitempty"`
	ServerType        string            `json:"server_type,omitempty"`
	Image             string            `json:"image,omitempty"`
	PublicIPv4        string            `json:"public_ipv4,omitempty"`
	PublicIPv6        string            `json:"public_ipv6,omitempty"`
	PrimaryIPv4ID     string            `json:"primary_ipv4_id,omitempty"`
	PrimaryIPv6ID     string            `json:"primary_ipv6_id,omitempty"`
	SSHKeyID          string            `json:"ssh_key_id,omitempty"`
	SSHKeyName        string            `json:"ssh_key_name,omitempty"`
	SSHKeyFingerprint string            `json:"ssh_key_fingerprint,omitempty"`
	FirewallID        string            `json:"firewall_id,omitempty"`
	FirewallName      string            `json:"firewall_name,omitempty"`
	Tags              map[string]string `json:"tags,omitempty"`
	CostProfile       CostProfileRef    `json:"cost_profile,omitempty"`
	CloudInitVersion  string            `json:"cloud_init_version,omitempty"`
}

type CostProfileRef struct {
	Provider             string `json:"provider"`
	Region               string `json:"region"`
	ServerType           string `json:"server_type"`
	Image                string `json:"image"`
	EstimatedHourlyCost  string `json:"estimated_hourly_cost,omitempty"`
	EstimatedMonthlyCost string `json:"estimated_monthly_cost,omitempty"`
	Caveat               string `json:"caveat,omitempty"`
}

type CleanupMetadata struct {
	GitHubRunnerID      int64    `json:"github_runner_id,omitempty"`
	ManagedPaths        []string `json:"managed_paths"`
	ProviderResourceIDs []string `json:"provider_resource_ids"`
	Notes               []string `json:"notes,omitempty"`
}

type OperationCheckpoint struct {
	Command   string    `json:"command"`
	Artifact  string    `json:"artifact"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SafetyMetadata struct {
	Code             string     `json:"code"`
	Allowed          bool       `json:"allowed"`
	SafetyProfile    string     `json:"safety_profile,omitempty"`
	Warnings         []string   `json:"warnings,omitempty"`
	AcceptedOverride string     `json:"accepted_override,omitempty"`
	AcceptedAt       *time.Time `json:"accepted_at,omitempty"`
}

func NewState() State {
	return State{SchemaVersion: SchemaVersion, Repositories: []RepositoryState{}}
}

func (s State) FindRepository(fullName string) (RepositoryState, bool) {
	for _, repo := range s.Repositories {
		if repo.Repo.FullName == fullName {
			return repo, true
		}
	}
	return RepositoryState{}, false
}

func (s State) ListRepositories() []RepositoryState {
	repos := make([]RepositoryState, len(s.Repositories))
	copy(repos, s.Repositories)
	return repos
}

func (s *State) UpdateRepository(repo RepositoryState) bool {
	for i, existing := range s.Repositories {
		if existing.Repo.FullName != repo.Repo.FullName {
			continue
		}
		if repo.CreatedAt.IsZero() {
			repo.CreatedAt = existing.CreatedAt
		}
		s.Repositories[i] = repo
		return true
	}
	return false
}

func (s *State) RemoveRepository(fullName string) bool {
	for i, repo := range s.Repositories {
		if repo.Repo.FullName != fullName {
			continue
		}
		s.Repositories = append(s.Repositories[:i], s.Repositories[i+1:]...)
		return true
	}
	return false
}

func (s *State) UpsertRepository(repo RepositoryState, replace bool) error {
	if s.SchemaVersion == "" {
		s.SchemaVersion = SchemaVersion
	}
	for i, existing := range s.Repositories {
		if existing.Repo.FullName != repo.Repo.FullName {
			continue
		}
		if !replace {
			return ErrRepositoryExists
		}
		if repo.CreatedAt.IsZero() {
			repo.CreatedAt = existing.CreatedAt
		}
		s.Repositories[i] = repo
		return nil
	}
	s.Repositories = append(s.Repositories, repo)
	return nil
}
