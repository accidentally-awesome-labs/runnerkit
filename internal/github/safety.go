package github

const (
	SafetyCodeOK         = "ok"
	SafetyCodePublicRisk = "public_repo_risk"
	SafetyCodeForkRisk   = "fork_risk"
	// AllowPublicRepoRiskFlag is the explicit danger override: --allow-public-repo-risk.
	AllowPublicRepoRiskFlag = "--allow-public-repo-risk"
)

const PublicRepoRiskTitle = "WARNING: Public repository risk"
const PublicRepoRiskBody = "Persistent self-hosted runners can execute untrusted workflow code from forks or public contributors."
const PublicRepoRiskNextAction = "Use GitHub-hosted runners, wait for RunnerKit ephemeral mode, or pass --allow-public-repo-risk after reviewing the risk."

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
