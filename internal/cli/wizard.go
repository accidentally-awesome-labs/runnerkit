package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
	rkstate "github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ux/nextaction"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ux/stage"
	"github.com/spf13/cobra"
)

func shouldOfferFirstRunWizard(store rkstate.Store) (bool, error) {
	repos, err := store.ListRepositories()
	if err != nil {
		return false, err
	}
	return len(repos) == 0, nil
}

func readLineTrim(r *bufio.Reader, out io.Writer, prompt string) (string, error) {
	if out != nil {
		_, _ = fmt.Fprint(out, prompt)
	}
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func runFirstRunWizard(ctx context.Context, cmd *cobra.Command, deps Dependencies, jsonOutput, noColor bool) error {
	renderer := newRenderer(deps, jsonOutput, noColor)
	store := rkstate.NewStore(deps.StateBaseDir)
	ok, err := shouldOfferFirstRunWizard(store)
	if err != nil {
		_ = renderer.Error("state_io_failed", "RunnerKit can't read saved runner state.", []string{"Check permissions for " + store.Path() + "."})
		return NewExitError(ExitStateIO, err)
	}
	if !ok {
		return cmd.Help()
	}
	if jsonOutput {
		na := []nextaction.Action{
			{ID: "wizard_non_interactive", Severity: nextaction.SeverityBlocking, Title: "First-run wizard requires a TTY for interactive prompts", Command: "runnerkit up --repo owner/name --host user@host", Kind: "run_local"},
		}
		p := nextaction.MergePayload(map[string]any{"ok": true, "command": "wizard"}, string(stage.NoLocalState), na)
		return renderer.JSON(p)
	}
	if !deps.TTY.StdinTTY || !deps.TTY.StdoutTTY {
		_ = renderer.Error("wizard_requires_tty", "The first-run wizard needs an interactive terminal.", []string{"Run `runnerkit up --repo owner/name --host user@host`, or set up with `runnerkit init` then `runnerkit register`."})
		return NewExitError(ExitInputRequired, errors.New("wizard requires tty"))
	}
	if deps.Prompts == nil {
		return cmd.Help()
	}

	br := bufio.NewReader(deps.In)
	choice, err := deps.Prompts.Select(ctx, ui.Prompt{Message: "Where do you want to run GitHub Actions jobs?"}, []ui.Option{
		{Value: "byo", Label: "BYO Linux machine (SSH)", Description: "Use your own server or desktop"},
		{Value: "cloud", Label: "Hetzner cloud (recommended default)", Description: "Provision a small VM via runnerkit up"},
	})
	if err != nil {
		return NewExitError(ExitInvalidInput, err)
	}
	if choice == "cloud" {
		_ = renderer.Step(1, 1, "Cloud setup", ui.Bullet("Run the guided cloud path (tokens via env / gh):"), ui.Next("runnerkit up --repo owner/name --cloud hetzner --mode persistent"), ui.Bullet("See docs/cloud-quickstart.md for credentials and sizing."))
		return nil
	}

	host, err := readLineTrim(br, deps.Out, "SSH target (user@host or user@host:port): ")
	if err != nil || host == "" {
		return NewExitError(ExitInvalidInput, fmt.Errorf("host required"))
	}
	repoRaw, err := readLineTrim(br, deps.Out, "GitHub repository (owner/name): ")
	if err != nil || repoRaw == "" {
		return NewExitError(ExitInvalidInput, fmt.Errorf("repo required"))
	}
	resolution, err := gh.ResolveTarget(ctx, gh.ResolveOptions{Repo: repoRaw, CommandRunner: deps.CommandRunner})
	if err != nil {
		_ = renderer.Error("invalid_repo", err.Error(), []string{"Use a repository you can access with current GitHub credentials."})
		return NewExitError(ExitInvalidInput, err)
	}

	one := fmt.Sprintf("runnerkit up --repo %s --host %s", resolution.Repo.FullName, host)
	box := ui.RenderBoxed(host, one, "This starts the full BYO install and registration flow from your workstation.", deps.UnicodeBox(), deps.TTY.Width)
	_, _ = fmt.Fprintln(deps.Out, box)
	_ = renderer.Step(2, 2, "Bookmark these commands", ui.Bullet("runnerkit register --repo NEW   # add another repo to the same host"), ui.Bullet("runnerkit status --all          # see saved runners"), ui.Bullet("runnerkit doctor --repo "+resolution.Repo.FullName+"   # diagnose issues"))
	return nil
}
