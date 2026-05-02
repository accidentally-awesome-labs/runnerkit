package github

const (
	SafetyCodeOK         = "ok"
	SafetyCodePublicRisk = "public_repo_risk"
	SafetyCodeForkRisk   = "fork_risk"
	// AllowPublicRepoRiskFlag is the explicit danger override: --allow-public-repo-risk.
	AllowPublicRepoRiskFlag = "--allow-public-repo-risk"
)

const PublicRepoRiskTitle = "WARNING: Public repository risk"

// PublicRepoRiskBody is the canonical user-facing description of the
// persistent-runner public/fork risk. Phase 5 tightened the wording
// because persistent runners are not "intended for trusted private
// repositories" — they are *unsafe* for any other workload.
const PublicRepoRiskBody = "Persistent self-hosted runners are unsafe for public, fork-based, or otherwise untrusted workflows."

// PublicRepoRiskNextAction recommends the safer ephemeral cloud command
// instead of waiting for ephemeral mode (which is now available) or
// silently telling users to pass --allow-public-repo-risk.
const PublicRepoRiskNextAction = "Use `runnerkit up --repo owner/name --mode ephemeral --cloud hetzner` for stronger isolation, or use GitHub-hosted runners."

// DangerousPersistentOverrideCopy is the explicit danger remediation
// shown alongside `--allow-public-repo-risk` so users cannot opt in to
// running untrusted code repeatedly on their machine without seeing the
// risk.
const DangerousPersistentOverrideCopy = "Only pass `--allow-public-repo-risk` if you accept that untrusted code can execute repeatedly on your machine."

type SafetyOptions struct {
	AllowPublicRepoRisk bool
}

type SafetyDecision struct {
	Allowed          bool
	Code             string
	Warnings         []string
	RequiresOverride string
}

func EvaluateSafety(repo Repo, opts SafetyOptions) SafetyDecision {
	if !repo.Private {
		return riskDecision(SafetyCodePublicRisk, opts.AllowPublicRepoRisk, []string{
			SafetyCodePublicRisk + ": " + PublicRepoRiskTitle,
			PublicRepoRiskBody,
			PublicRepoRiskNextAction,
		})
	}
	if repo.Fork {
		return riskDecision(SafetyCodeForkRisk, false, []string{
			SafetyCodeForkRisk + ": Fork repository risk",
			"Persistent self-hosted runners for forks require explicit review before setup continues.",
		})
	}
	return SafetyDecision{Allowed: true, Code: SafetyCodeOK}
}

func riskDecision(code string, allowed bool, warnings []string) SafetyDecision {
	decision := SafetyDecision{Allowed: allowed, Code: code, Warnings: warnings}
	if !allowed {
		decision.RequiresOverride = AllowPublicRepoRiskFlag
	}
	return decision
}
