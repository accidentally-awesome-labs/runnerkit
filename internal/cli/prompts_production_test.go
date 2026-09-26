package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
	"github.com/accidentally-awesome-labs/runnerkit/internal/provider"
	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/accidentally-awesome-labs/runnerkit/internal/testsupport"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
)

// A-02 (P1-2): typed confirmations must work with the production
// prompter, not only with test fakes that happen to implement Input.
// v1.3.x's CLIPrompter had no Input method, so `destroy` in a real
// terminal always failed with "requires typed confirmation".

func executeDestroyWithProductionPrompter(t *testing.T, stdin string) (*provider.FakeProvider, string, string, error) {
	t.Helper()
	stateDir := t.TempDir()
	repo := saveCloudStateForDestroy(t, stateDir)
	github := &testsupport.GitHubService{Runners: []gh.Runner{testsupport.HealthyRunner()}}
	cloud := &provider.FakeProvider{VerifyOut: provider.VerificationResult{OK: true}}
	var out, errOut, promptOut bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &out,
		Err:            &errOut,
		StateBaseDir:   stateDir,
		GitHub:         github,
		RemoteExecutor: doctorRemote(true),
		Providers:      provider.NewRegistry(cloud),
		Prompts:        ui.NewCLIPrompter(strings.NewReader(stdin), &promptOut),
		TTY:            ui.TerminalCapabilities{StdinTTY: true, StdoutTTY: true, Width: 80},
		CommandRunner:  staticCommandRunner{remote: "git@github.com:owner/repo.git"},
		Sleep:          noSleep,
	})
	cmd.SetArgs([]string{"destroy", "--repo", repo.Repo.FullName, "--no-color"})
	err := cmd.Execute()
	if !strings.Contains(promptOut.String(), "destroy owner/repo") {
		t.Fatalf("production prompter did not render the typed-confirmation prompt: %q", promptOut.String())
	}
	if _, found, loadErr := state.NewStore(stateDir).GetRepository(repo.Repo.FullName); loadErr != nil {
		t.Fatalf("load state: %v", loadErr)
	} else if err == nil && found {
		t.Fatalf("destroy succeeded but state was kept")
	}
	return cloud, out.String(), errOut.String(), err
}

func TestProductionPrompter_DestroyAcceptsTypedPhrase(t *testing.T) {
	cloud, out, errOut, err := executeDestroyWithProductionPrompter(t, "destroy owner/repo\n")
	if err != nil {
		t.Fatalf("destroy with production prompter returned error: %v\nstdout=%s\nstderr=%s", err, out, errOut)
	}
	if cloud.DestroyCalls != 1 {
		t.Fatalf("expected one provider Destroy call, got %d", cloud.DestroyCalls)
	}
}

func TestProductionPrompter_DestroyWrongPhraseMakesNoProviderCalls(t *testing.T) {
	cloud, out, errOut, err := executeDestroyWithProductionPrompter(t, "destroy someone/else\n")
	if err == nil {
		t.Fatalf("expected wrong phrase to fail\nstdout=%s\nstderr=%s", out, errOut)
	}
	if ExitCode(err) == ExitSuccess {
		t.Fatalf("wrong phrase exited 0")
	}
	if cloud.DestroyCalls != 0 || cloud.VerifyDestroyedCalls != 0 || cloud.ProvisionCalls != 0 {
		t.Fatalf("wrong phrase reached the provider: %#v", cloud)
	}
	if !strings.Contains(errOut, "Canceled") {
		t.Fatalf("expected cancel message on stderr: %s", errOut)
	}
}

func TestProductionPrompter_BYOHostEntryAcceptsUserAtHost(t *testing.T) {
	var promptOut bytes.Buffer
	deps := normalizeDependencies(Dependencies{
		StateBaseDir: t.TempDir(),
		Prompts:      ui.NewCLIPrompter(strings.NewReader("alice@example.com\n"), &promptOut),
		TTY:          ui.TerminalCapabilities{StdinTTY: true, StdoutTTY: true, Width: 80},
	})
	renderer := newRenderer(deps, false, true)
	target, err := resolveBYOTarget(context.Background(), deps, renderer, &upOptions{}, false)
	if err != nil {
		t.Fatalf("resolveBYOTarget with production prompter returned error: %v", err)
	}
	if target.User != "alice" || target.Host != "example.com" {
		t.Fatalf("unexpected target %#v", target)
	}
	if !strings.Contains(promptOut.String(), "SSH target (user@host):") {
		t.Fatalf("missing SSH target prompt: %q", promptOut.String())
	}
}
