package cli

import (
	"io"
	"net/http"
	"os"
	"time"

	gh "github.com/salar/runnerkit/internal/github"
	"github.com/salar/runnerkit/internal/redact"
	"github.com/salar/runnerkit/internal/ui"
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
	// GitHubBaseURL string is a test-only GitHub API base URL override.
	GitHubBaseURL    string
	GitHubHTTPClient *http.Client
	StateBaseDir     string
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
		deps.GitHub = gh.NewService(gh.ServiceOptions{CommandRunner: deps.CommandRunner, Env: deps.GitHubEnv, BaseURL: deps.GitHubBaseURL, HTTPClient: deps.GitHubHTTPClient})
	}
	return deps
}

// NewRootCommand constructs the runnerkit command tree.
func NewRootCommand(deps Dependencies) *cobra.Command {
	deps = normalizeDependencies(deps)

	var jsonOutput bool
	var noColor bool

	root := &cobra.Command{Use: "runnerkit"}
	root.Short = "Prepare and manage GitHub Actions self-hosted runners"
	root.Long = "RunnerKit prepares and manages GitHub Actions self-hosted runners from a CLI-first workflow."
	root.SilenceUsage = true
	root.SilenceErrors = true
	root.SetIn(deps.In)
	root.SetOut(deps.Out)
	root.SetErr(deps.Err)
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return NewExitError(ExitInvalidInput, err)
	})

	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "write machine-readable JSON to stdout")
	root.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable ANSI color output")

	root.AddCommand(newVersionCommand(deps, &jsonOutput, &noColor))
	root.AddCommand(newUpCommand(deps, &jsonOutput, &noColor))
	root.AddCommand(newStateCommand(deps, &jsonOutput, &noColor))

	return root
}

func newVersionCommand(deps Dependencies, jsonOutput *bool, noColor *bool) *cobra.Command {
	cmd := &cobra.Command{Use: "version"}
	cmd.Short = "Print version information"
	cmd.RunE = func(_ *cobra.Command, _ []string) error {
		renderer := newRenderer(deps, *jsonOutput, *noColor)
		if *jsonOutput {
			return renderer.JSON(map[string]any{
				"ok":      true,
				"command": "version",
				"version": deps.Version,
			})
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
