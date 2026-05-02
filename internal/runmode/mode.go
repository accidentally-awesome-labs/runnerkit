// Package runmode owns mode and safety-profile decisions for runnerkit up.
//
// It defines the canonical persistent vs ephemeral mode constants, the four
// safety profile constants used to gate setup-path mutation, the typed
// tradeoff payloads rendered in human and JSON output before any GitHub,
// remote, provider, or state side effect, and the default ephemeral TTL.
//
// runmode does not perform any I/O. It only normalizes user-supplied input
// and computes a Decision describing why one mode/profile is being chosen.
package runmode

import (
	"errors"
	"strings"
	"time"

	gh "github.com/salar/runnerkit/internal/github"
)

// Canonical mode and safety profile values surfaced to users in
// `--mode` flags, prompts, JSON output, and persisted state.
const (
	ModePersistent = "persistent"
	ModeEphemeral  = "ephemeral"

	ProfilePersistentTrusted = "persistent-trusted"
	ProfilePersistentRisky   = "persistent-risky"
	ProfileEphemeralBYO      = "ephemeral-byo"
	ProfileEphemeralCloud    = "ephemeral-cloud"

	// DefaultEphemeralTTL is the safeguard timeout the ephemeral lifecycle
	// will use to finalize a runner when no job completes. Phase 5 plans
	// 05-02/05-03 wire this into the bootstrap timer; 05-01 only renders it
	// in user-facing tradeoff output.
	DefaultEphemeralTTL = 24 * time.Hour
)

// Tradeoffs describes the persistent vs ephemeral cost/isolation/cleanup/
// operations/log preservation tradeoff that must appear in human and JSON
// output before any setup-path mutation runs. Strings are stable and
// asserted against in tests.
type Tradeoffs struct {
	Cost       string `json:"cost"`
	Isolation  string `json:"isolation"`
	Cleanup    string `json:"cleanup"`
	Operations string `json:"operations"`
	Logs       string `json:"logs"`
}

// Decision describes the chosen mode and the safety implications of that
// choice for a specific repository. It is the single payload used by the
// CLI to render tradeoffs and by the safety gate to decide whether to
// continue, prompt, or block before mutation.
type Decision struct {
	Mode                         string    `json:"mode"`
	SafetyProfile                string    `json:"safety_profile"`
	Tradeoffs                    Tradeoffs `json:"tradeoffs"`
	RecommendedFor               []string  `json:"recommended_for"`
	NotRecommendedFor            []string  `json:"not_recommended_for"`
	Warnings                     []string  `json:"warnings"`
	RequiresTypedAcknowledgement string    `json:"requires_typed_acknowledgement,omitempty"`
	CleanupCommand               string    `json:"cleanup_command,omitempty"`
}

// Options carries CLI-level decision inputs that runmode does not own
// itself: the user-supplied mode, the resolved setup path ("byo" or
// "cloud"), and explicit risk overrides. SetupPath is informative; the
// safety profile only depends on Mode and repo trust state for now.
type Options struct {
	Mode                  string
	SetupPath             string
	AllowPersistentRisk   bool
	AllowEphemeralBYORisk bool
}

// Tradeoff strings exposed as constants so the CLI can use the same exact
// copy in multiple render paths and tests can assert without duplication.
const (
	tradeoffPersistentCost       = "Lowest ongoing setup cost for repeated trusted private jobs; cloud resources keep billing until cleanup."
	tradeoffPersistentIsolation  = "Reuses one runner across jobs, so workflow code can persist on the machine."
	tradeoffPersistentCleanup    = "Requires runnerkit down for BYO or runnerkit destroy for cloud."
	tradeoffPersistentOperations = "Best for trusted private solo-development workflows that run repeatedly."
	tradeoffPersistentLogs       = "Live runner _diag and systemd logs remain on the machine until cleanup."

	tradeoffEphemeralCost       = "Higher setup and cleanup cost per run; cloud resources keep billing until destroy verifies cleanup."
	tradeoffEphemeralIsolation  = "GitHub assigns at most one job then deregisters the runner; BYO hosts are still reused machines."
	tradeoffEphemeralCleanup    = "TTL finalizer preserves logs and cleanup still uses down for BYO or destroy for cloud."
	tradeoffEphemeralOperations = "One scoped runner only; not autoscaling or a fleet manager."
	tradeoffEphemeralLogs       = "RunnerKit preserves best-effort _diag and systemd journal logs before cleanup."

	// WarningPublicForkPersistent is the user-facing public/fork persistent
	// warning. The CLI safety gate also surfaces this through internal/github
	// and tests assert exact copy in 05-01-03.
	WarningPublicForkPersistent = "Persistent self-hosted runners are unsafe for public, fork-based, or otherwise untrusted workflows."

	// WarningEphemeralBYONotCleanVM warns BYO ephemeral users that the
	// machine is not a clean VM and unrelated secrets must not live there.
	WarningEphemeralBYONotCleanVM = "BYO ephemeral mode is a one-job GitHub registration, not a clean virtual machine. Do not store unrelated secrets on the host."

	// WarningEphemeralCloudBillable reminds users that ephemeral cloud
	// runners still incur Hetzner charges until destroy verifies cleanup.
	WarningEphemeralCloudBillable = "Ephemeral cloud runners still create billable Hetzner resources. Billing stops only after `runnerkit destroy --repo owner/name` verifies cleanup."

	// WarningEphemeralNotFleet keeps users from expecting autoscaling.
	WarningEphemeralNotFleet = "Ephemeral mode is not a fleet manager. RunnerKit creates one scoped runner; jobs with matching labels can still queue if no runner is online."
)

// errInvalidMode is returned by Normalize for any value that is not
// persistent, ephemeral, or empty. The wrapped message contains the exact
// supported-modes copy users see in CLI errors.
var errInvalidMode = errors.New("invalid mode. Supported modes: --mode persistent or --mode ephemeral.")

// Normalize lower-cases and trims the user-supplied --mode value. Empty
// and persistent normalize to ModePersistent. ephemeral normalizes to
// ModeEphemeral. Anything else returns an error containing the supported
// modes copy so the CLI can surface the same error message everywhere.
func Normalize(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "", ModePersistent:
		return ModePersistent, nil
	case ModeEphemeral:
		return ModeEphemeral, nil
	default:
		return "", errInvalidMode
	}
}

// Evaluate computes the safety profile and tradeoff payload for the chosen
// repo and CLI options. It does not block on its own; callers compose it
// with internal/github safety enforcement to decide whether to continue.
func Evaluate(repo gh.Repo, opts Options) Decision {
	mode, err := Normalize(opts.Mode)
	if err != nil {
		mode = ModePersistent
	}
	profile := profileFor(mode, repo, opts)
	decision := Decision{Mode: mode, SafetyProfile: profile, Tradeoffs: tradeoffsFor(mode)}
	switch profile {
	case ProfilePersistentTrusted:
		decision.RecommendedFor = []string{"trusted private repositories", "repeated solo-development workflows"}
		decision.NotRecommendedFor = []string{"public repositories", "forks", "untrusted workflows"}
		decision.CleanupCommand = "runnerkit down --repo " + repo.FullName
	case ProfilePersistentRisky:
		decision.RecommendedFor = []string{}
		decision.NotRecommendedFor = []string{"public repositories", "forks", "untrusted workflows"}
		decision.Warnings = append(decision.Warnings, WarningPublicForkPersistent)
		decision.RequiresTypedAcknowledgement = "allow persistent public repo risk for " + repo.FullName
		decision.CleanupCommand = "runnerkit down --repo " + repo.FullName
	case ProfileEphemeralBYO:
		decision.RecommendedFor = []string{"trusted private workflows that want one-job isolation"}
		decision.NotRecommendedFor = []string{"unrelated secrets on the host", "long-running queues without a runner online"}
		decision.Warnings = append(decision.Warnings, WarningEphemeralBYONotCleanVM, WarningEphemeralNotFleet)
		decision.CleanupCommand = "runnerkit down --repo " + repo.FullName
	case ProfileEphemeralCloud:
		decision.RecommendedFor = []string{"public, fork, or otherwise untrusted workflows", "stronger isolation per job"}
		decision.NotRecommendedFor = []string{"long-running queues without a runner online", "fleet/autoscaling expectations"}
		decision.Warnings = append(decision.Warnings, WarningEphemeralCloudBillable, WarningEphemeralNotFleet)
		decision.CleanupCommand = "runnerkit destroy --repo " + repo.FullName
	}
	return decision
}

func profileFor(mode string, repo gh.Repo, opts Options) string {
	switch mode {
	case ModeEphemeral:
		if strings.EqualFold(strings.TrimSpace(opts.SetupPath), "cloud") {
			return ProfileEphemeralCloud
		}
		return ProfileEphemeralBYO
	default:
		if !repo.Private || repo.Fork {
			return ProfilePersistentRisky
		}
		return ProfilePersistentTrusted
	}
}

func tradeoffsFor(mode string) Tradeoffs {
	if mode == ModeEphemeral {
		return Tradeoffs{
			Cost:       tradeoffEphemeralCost,
			Isolation:  tradeoffEphemeralIsolation,
			Cleanup:    tradeoffEphemeralCleanup,
			Operations: tradeoffEphemeralOperations,
			Logs:       tradeoffEphemeralLogs,
		}
	}
	return Tradeoffs{
		Cost:       tradeoffPersistentCost,
		Isolation:  tradeoffPersistentIsolation,
		Cleanup:    tradeoffPersistentCleanup,
		Operations: tradeoffPersistentOperations,
		Logs:       tradeoffPersistentLogs,
	}
}
