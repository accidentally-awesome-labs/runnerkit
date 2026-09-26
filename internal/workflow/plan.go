package workflow

import (
	"context"
	"fmt"
)

type StepStatus string

const (
	StepPending StepStatus = "pending"
	StepDone    StepStatus = "done"
	StepSkipped StepStatus = "skipped"
	StepFailed  StepStatus = "failed"
)

const (
	StepResolveRepo  = "resolve_repo"
	StepVerifyAuth   = "verify_auth"
	StepCheckSafety  = "check_safety"
	StepPreviewState = "preview_state"
	StepSaveState    = "save_state"

	StepFixDependencies  = "fix_dependencies"
	StepCreateRunnerUser = "create_runner_user"
	StepSetupRunnerImage = "setup_runner_image"
	StepDownloadRunner   = "download_runner"
	StepConfigureRunner  = "configure_runner"
	StepInstallService   = "install_service"
	StepVerifyService    = "verify_service"
)

type Checkpoint struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Prompt   string `json:"prompt"`
	Required bool   `json:"required"`
}

type Step struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Status      StepStatus  `json:"status"`
	Checkpoint  *Checkpoint `json:"checkpoint,omitempty"`
	Warnings    []string    `json:"warnings,omitempty"`
}

type Plan struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Steps    []Step   `json:"steps"`
	Warnings []string `json:"warnings,omitempty"`
}

type StepRunner interface {
	RunStep(ctx context.Context, step Step) (StepStatus, error)
}

type StepRunnerFunc func(ctx context.Context, step Step) (StepStatus, error)

func (f StepRunnerFunc) RunStep(ctx context.Context, step Step) (StepStatus, error) {
	return f(ctx, step)
}

func FoundationUpPlan() Plan {
	return Plan{
		ID:    "foundation_up",
		Title: "Prepare RunnerKit foundation state",
		Steps: []Step{
			{ID: StepResolveRepo, Name: "Resolve repository", Status: StepPending},
			{ID: StepVerifyAuth, Name: "Verify GitHub auth", Status: StepPending},
			{ID: StepCheckSafety, Name: "Check safety", Status: StepPending},
			{ID: StepPreviewState, Name: "Preview state", Status: StepPending},
			{ID: StepSaveState, Name: "Save state", Status: StepPending, Checkpoint: &Checkpoint{ID: "confirm_save_state", Type: "confirm", Prompt: "Save this foundation state? [y/N]", Required: true}},
		},
	}
}

func BootstrapPlan() Plan {
	return Plan{
		ID:    "bootstrap-plan",
		Title: "Apply BYO runner install plan",
		Steps: []Step{
			{ID: StepFixDependencies, Name: "Fix dependencies", Status: StepPending},
			{ID: StepCreateRunnerUser, Name: "Create runner user", Status: StepPending},
			{ID: StepDownloadRunner, Name: "Download runner", Status: StepPending},
			{ID: StepConfigureRunner, Name: "Configure runner", Status: StepPending},
			{ID: StepInstallService, Name: "Install service", Status: StepPending},
			{ID: StepVerifyService, Name: "Verify service", Status: StepPending},
		},
	}
}

// SetupRunnerImageName is the plan text for setup_runner_image. It is the
// consent point for what the step does as root, so it names the Docker
// group grant and the third-party apt sources.
const SetupRunnerImageName = "Set up runner image: install Docker CE, language runtimes and browsers from third-party apt sources (Docker, NodeSource, Microsoft, Google Chrome, Mozilla PPA, GitHub CLI) and add the runner user to the docker group (root-equivalent for every job)"

// BootstrapPlanWithImageSetup is BootstrapPlan with setup_runner_image
// after create_runner_user, matching the order bootstrap.Apply runs on
// Ubuntu/Debian hosts.
func BootstrapPlanWithImageSetup() Plan {
	plan := BootstrapPlan()
	steps := make([]Step, 0, len(plan.Steps)+1)
	for _, step := range plan.Steps {
		steps = append(steps, step)
		if step.ID == StepCreateRunnerUser {
			steps = append(steps, Step{ID: StepSetupRunnerImage, Name: SetupRunnerImageName, Status: StepPending})
		}
	}
	plan.Steps = steps
	return plan
}

// HasStep reports whether the plan contains a step with the given ID.
func (p Plan) HasStep(id string) bool {
	for _, step := range p.Steps {
		if step.ID == id {
			return true
		}
	}
	return false
}

func (p Plan) Checklist() []string {
	items := make([]string, 0, len(p.Steps))
	for _, step := range p.Steps {
		items = append(items, fmt.Sprintf("%s %s - %s", statusMarker(step.Status), step.ID, step.Name))
	}
	return items
}

func Apply(ctx context.Context, plan Plan, runner StepRunner) (Plan, error) {
	if runner == nil {
		runner = StepRunnerFunc(func(context.Context, Step) (StepStatus, error) { return StepDone, nil })
	}
	for i, step := range plan.Steps {
		if err := ctx.Err(); err != nil {
			plan.Steps[i].Status = StepFailed
			return plan, err
		}
		if step.Status == "" {
			step.Status = StepPending
		}
		if step.Status == StepDone || step.Status == StepSkipped {
			plan.Steps[i] = step
			continue
		}
		status, err := runner.RunStep(ctx, step)
		if status == "" {
			status = StepDone
		}
		step.Status = status
		if err != nil {
			step.Status = StepFailed
			plan.Steps[i] = step
			return plan, err
		}
		plan.Steps[i] = step
	}
	return plan, nil
}

func statusMarker(status StepStatus) string {
	switch status {
	case StepDone:
		return "[x]"
	case StepSkipped:
		return "[-]"
	case StepFailed:
		return "[!]"
	default:
		return "[ ]"
	}
}
