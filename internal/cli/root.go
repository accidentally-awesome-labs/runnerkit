package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
	"github.com/accidentally-awesome-labs/runnerkit/internal/provider"
	"github.com/accidentally-awesome-labs/runnerkit/internal/provider/hetzner"
	"github.com/accidentally-awesome-labs/runnerkit/internal/redact"
	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	"github.com/accidentally-awesome-labs/runnerkit/internal/rklog"
	rkstate "github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ux/nextaction"
	"github.com/spf13/cobra"
)

// Dependencies are injectable command dependencies used by tests and main.
type Dependencies struct {
	Version       string
	In            io.Reader
	Out           io.Writer
	Err           io.Writer
	TTY           ui.TerminalCapabilities
	Prompts       ui.Prompter
	Clock         func() time.Time
	CommandRunner gh.CommandRunner
	GitHub        GitHubService
	GitHubEnv     map[string]string
	Providers     provider.Registry
	// GitHubBaseURL string is a test-only GitHub API base URL override.
	GitHubBaseURL    string
	GitHubHTTPClient *http.Client
	StateBaseDir     string
	RemoteExecutor   remote.Executor
	// Logger is optional structured logging (slog). When nil, NewFromEnv
	// uses RUNNERKIT_LOG against stderr — see package rklog.
	Logger       *slog.Logger
	PollInterval time.Duration
	PollTimeout  time.Duration
	Sleep        func(context.Context, time.Duration) error
	// Explain, when non-nil and returns true, prints WHY/RUNS/TAKES blocks for supported commands.
	Explain func() bool
	// UnicodeBox, when non-nil and returns true, uses UTF-8 borders in boxed command output.
	UnicodeBox func() bool
}

func normalizeDependencies(deps Dependencies) Dependencies {
	if deps.Version == "" {
		deps.Version = "dev"
	}
	if deps.In == nil {
		deps.In = io.Reader(nil)
	}
	if deps.Out == nil {
		deps.Out = io.Discard
	}
	if deps.Err == nil {
		deps.Err = io.Discard
	}
	if deps.Logger == nil {
		deps.Logger = rklog.NewFromEnv(deps.Err)
	}
	// config.json (doctor --ignore) and sessions/ (BYO checklists)
	// resolve relative to StateBaseDir. Leaving it "" made both land in
	// the process CWD, often the user's repository checkout (P1-3).
	if strings.TrimSpace(deps.StateBaseDir) == "" {
		deps.StateBaseDir = rkstate.DefaultBaseDir()
	}
	if deps.Clock == nil {
		deps.Clock = time.Now
	}
	if deps.TTY.Width == 0 {
		deps.TTY.Width = 80
	}
	if deps.CommandRunner == nil {
		deps.CommandRunner = gh.OSCommandRunner{}
	}
	if deps.GitHub == nil {
		deps.GitHub = gh.NewService(gh.ServiceOptions{CommandRunner: deps.CommandRunner, Env: deps.GitHubEnv, BaseURL: deps.GitHubBaseURL, HTTPClient: deps.GitHubHTTPClient, Logger: deps.Logger})
	}
	if deps.Providers == nil {
		deps.Providers = provider.NewRegistry(hetzner.NewProvider(nil, hetzner.WithLogger(deps.Logger)))
	}
	if deps.RemoteExecutor == nil {
		deps.RemoteExecutor = remote.NewSystemExecutor()
	}
	deps.RemoteExecutor = remote.WrapWithLogging(deps.RemoteExecutor, deps.Logger, nil)
	if deps.PollInterval == 0 {
		deps.PollInterval = defaultRunnerPollInterval
	}
	if deps.PollTimeout == 0 {
		deps.PollTimeout = defaultRunnerPollTimeout
	}
	if deps.Sleep == nil {
		deps.Sleep = func(ctx context.Context, d time.Duration) error {
			timer := time.NewTimer(d)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		}
	}
	if deps.Explain == nil {
		deps.Explain = func() bool { return false }
	}
	if deps.UnicodeBox == nil {
		deps.UnicodeBox = func() bool { return false }
	}
	return deps
}

// NewRootCommand constructs the runnerkit command tree.
func NewRootCommand(deps Dependencies) *cobra.Command {
	deps = normalizeDependencies(deps)

	var jsonOutput bool
	var noColor bool
	var explain bool
	var unicodeBox bool

	root := &cobra.Command{Use: "runnerkit"}
	root.Short = "Prepare and manage GitHub Actions self-hosted runners"
	root.Long = "RunnerKit prepares and manages GitHub Actions self-hosted runners from a CLI-first workflow."
	root.SilenceUsage = true
	root.SilenceErrors = true
	root.SetIn(deps.In)
	root.SetOut(deps.Out)
	root.SetErr(deps.Err)
	// --version prints exactly "runnerkit <version>" (P1-1: it used to be a
	// silent no-op because Version was never set).
	root.Version = deps.Version
	root.SetVersionTemplate("runnerkit {{.Version}}\n")
	// Unknown subcommands exit 2 like other usage errors. Cobra's own check
	// (legacyArgs) returns a plain error that maps to exit 1; keep its wording
	// and "Did you mean this?" suggestions.
	root.SuggestionsMinimumDistance = 2
	root.Args = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		var suggestions strings.Builder
		if names := cmd.SuggestionsFor(args[0]); len(names) > 0 {
			suggestions.WriteString("\n\nDid you mean this?\n")
			for _, name := range names {
				suggestions.WriteString("\t" + name + "\n")
			}
		}
		return NewExitError(ExitInvalidInput, fmt.Errorf("unknown command %q for %q%s", args[0], cmd.CommandPath(), suggestions.String()))
	}
	// SilenceErrors stays on so each failure is printed exactly once: flag
	// errors are rendered here, command errors by their RunE, and anything
	// left unrendered by cmd/runnerkit via ReportUnrenderedError.
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		asJSON := jsonOutput || argsRequestJSON(os.Args[1:])
		message := "invalid_flag: " + err.Error()
		if asJSON {
			message = err.Error()
		}
		renderer := newRenderer(deps, asJSON, noColor)
		_ = renderer.Error("invalid_flag", message, []string{"Run '" + cmd.CommandPath() + " --help' for usage."})
		return NewExitError(ExitInvalidInput, err)
	})
	root.PersistentPreRun = func(cmd *cobra.Command, _ []string) {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		if deps.Logger != nil && deps.Logger.Enabled(ctx, slog.LevelInfo) {
			deps.Logger.InfoContext(ctx, "runnerkit.cli.begin",
				slog.String("command", cmd.CommandPath()),
				slog.String("version", deps.Version),
			)
		}
	}

	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "write machine-readable JSON to stdout")
	root.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable ANSI color output")
	root.PersistentFlags().BoolVar(&explain, "explain", false, "print WHY/RUNS/TAKES context before major steps (supported commands)")
	root.PersistentFlags().BoolVar(&unicodeBox, "unicode", false, "use UTF-8 box-drawing characters for command panels")

	deps.Explain = func() bool { return explain }
	deps.UnicodeBox = func() bool { return unicodeBox }

	root.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		return runFirstRunWizard(ctx, cmd, deps, jsonOutput, noColor)
	}

	root.AddCommand(newVersionCommand(deps, &jsonOutput, &noColor))
	root.AddCommand(newInitCommand(deps, &jsonOutput, &noColor))
	root.AddCommand(newRegisterCommand(deps, &jsonOutput, &noColor))
	root.AddCommand(newUpCommand(deps, &jsonOutput, &noColor))
	root.AddCommand(newListCommand(deps, &jsonOutput, &noColor))
	root.AddCommand(newStatusCommand(deps, &jsonOutput, &noColor))
	root.AddCommand(newLogsCommand(deps, &jsonOutput, &noColor))
	root.AddCommand(newDoctorCommand(deps, &jsonOutput, &noColor))
	root.AddCommand(newRecoverCommand(deps, &jsonOutput, &noColor))
	root.AddCommand(newDownCommand(deps, &jsonOutput, &noColor))
	root.AddCommand(newDestroyCommand(deps, &jsonOutput, &noColor))
	root.AddCommand(newStateCommand(deps, &jsonOutput, &noColor))
	root.AddCommand(newUpgradeCommand(deps, &jsonOutput, &noColor))
	root.AddCommand(newUpgradeRunnerCommand(deps, &jsonOutput, &noColor))
	root.AddCommand(newRemovedByoPrepareCommand(deps, &noColor))

	return root
}

func newVersionCommand(deps Dependencies, jsonOutput *bool, noColor *bool) *cobra.Command {
	cmd := &cobra.Command{Use: "version"}
	cmd.Short = "Print version information"
	cmd.RunE = func(_ *cobra.Command, _ []string) error {
		renderer := newRenderer(deps, *jsonOutput, *noColor)
		if *jsonOutput {
			p := nextaction.MergePayload(map[string]any{
				"ok":      true,
				"command": "version",
				"version": deps.Version,
			}, "", nil)
			return renderer.JSON(p)
		}
		return renderer.Step(1, 1, "Version", ui.Success("RunnerKit "+deps.Version))
	}
	return cmd
}

func newRenderer(deps Dependencies, jsonOutput bool, noColor bool) *ui.Renderer {
	format := ui.FormatHuman
	if jsonOutput {
		format = ui.FormatJSON
	}
	caps := deps.TTY
	if caps.Width == 0 {
		caps.Width = 80
	}
	if noColor || os.Getenv("NO_COLOR") != "" || os.Getenv("CLICOLOR") == "0" || os.Getenv("TERM") == "dumb" {
		caps.Color = false
	}
	if os.Getenv("TERM") == "dumb" || !caps.StdoutTTY {
		caps.ASCII = true
	}
	return ui.NewRenderer(deps.Out, deps.Err, format, caps, redact.New())
}
