package cli

import (
	"bytes"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ops"
	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	rkstate "github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/accidentally-awesome-labs/runnerkit/internal/testsupport"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
)

// forbiddenLifecycleScript matches the remote actions only `up` and
// `register` may perform: configuring a runner, deleting its credentials,
// or installing its systemd service (A-06). A bare `config\.sh` pattern
// would wrongly flag `down`/`destroy`, which legitimately send
// `config.sh remove` (RenderRemoveConfigScript) and `svc.sh uninstall`.
var forbiddenLifecycleScript = regexp.MustCompile(`config\.sh --unattended|rm -f [^\n]*\.credentials|svc\.sh install`)

// lifecycleInvariantArgs lists, per subcommand, the invocations to exercise,
// including every flag combination that used to reach a lifecycle mutator.
// Every registered subcommand except up/register must appear here, so a new
// command cannot skip the invariant silently.
var lifecycleInvariantArgs = map[string][][]string{
	"":               {{}},
	"version":        {{"version"}},
	"init":           {{"init"}, {"init", "--print-install-command"}},
	"list":           {{"list"}},
	"status":         {{"status", "--repo", testsupport.TestRepoFullName}},
	"logs":           {{"logs", "--repo", testsupport.TestRepoFullName}},
	"doctor":         {{"doctor", "--repo", testsupport.TestRepoFullName, "--deep"}, {"doctor", "--repo", testsupport.TestRepoFullName, "--fix", "--yes"}},
	"recover":        {{"recover", "--repo", testsupport.TestRepoFullName, "--yes"}, {"recover", "--repo", testsupport.TestRepoFullName, "--restart-service", "--yes"}, {"recover", "--repo", testsupport.TestRepoFullName, "--reinstall-service", "--yes"}, {"recover", "--repo", testsupport.TestRepoFullName, "--reregister", "--yes"}},
	"down":           {{"down", "--repo", testsupport.TestRepoFullName, "--yes"}, {"unregister", "--repo", testsupport.TestRepoFullName, "--yes"}},
	"destroy":        {{"destroy", "--repo", testsupport.TestRepoFullName, "--yes"}},
	"state":          {{"state", "show", "--repo", testsupport.TestRepoFullName}},
	"upgrade":        {{"upgrade"}},
	"upgrade-runner": {{"upgrade-runner", "--repo", testsupport.TestRepoFullName, "--yes"}, {"upgrade-runner", "--repo", testsupport.TestRepoFullName, "--yes", "--force"}},
	"byo-prepare":    {{"byo-prepare", "--host", "alice@example.com"}},
}

// lifecycleInvariantExempt are the only commands allowed to configure a
// runner or install its service.
var lifecycleInvariantExempt = map[string]bool{"up": true, "register": true}

func newLifecycleInvariantRoot(t *testing.T, repo rkstate.RepositoryState) (*testsupport.RemoteExecutor, func(args []string)) {
	t.Helper()
	stateDir := seedRepoState(t, repo)
	// A recording executor whose host key matches the seeded state, so
	// commands get past the host-key gate and reach their mutating paths.
	remoteExec := &testsupport.RemoteExecutor{
		ProbeHostKeyResult: remote.HostKey{Fingerprint: "SHA256:fakehostfingerprint"},
		Results: map[string]remote.Result{
			ops.CommandStatusSystemdShow: {Stdout: "LoadState=loaded\nActiveState=failed\nSubState=failed\nUnitFileState=enabled\nExecMainStatus=1\n", ExitCode: 0},
			"resolve.runner_unit.show":   {Stdout: "LoadState=loaded\n", ExitCode: 0},
		},
	}
	github := &testsupport.GitHubService{
		Runners:           []gh.Runner{testsupport.HealthyRunner()},
		RemovalToken:      gh.RunnerToken{Token: "invariant-removal-token", ExpiresAt: time.Now().Add(time.Hour)},
		RegistrationToken: gh.RunnerToken{Token: "invariant-registration-token", ExpiresAt: time.Now().Add(time.Hour)},
	}
	run := func(args []string) {
		var out, errOut bytes.Buffer
		in := strings.NewReader("")
		cmd := NewRootCommand(Dependencies{
			Version:        "test-version",
			In:             in,
			Out:            &out,
			Err:            &errOut,
			StateBaseDir:   stateDir,
			GitHub:         github,
			RemoteExecutor: remoteExec,
			CommandRunner:  staticCommandRunner{remote: "git@github.com:owner/repo.git"},
			Prompts:        ui.NewCLIPrompter(in, &out),
			Sleep:          noSleep,
		})
		cmd.SetArgs(append([]string{"--no-color"}, args...))
		// Exit status is irrelevant here: refusals and failures are fine,
		// only the scripts sent to the host matter.
		_ = cmd.Execute()
	}
	return remoteExec, run
}

func TestLifecycleInvariant_OnlyUpAndRegisterConfigureRunners(t *testing.T) {
	root := NewRootCommand(Dependencies{StateBaseDir: t.TempDir()})
	var missing []string
	for _, sub := range root.Commands() {
		name := sub.Name()
		if lifecycleInvariantExempt[name] || name == "help" || name == "completion" {
			continue
		}
		if _, ok := lifecycleInvariantArgs[name]; !ok {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("add lifecycle invariant rows for new subcommands: %v", missing)
	}

	sentRemove := false
	for name, rows := range lifecycleInvariantArgs {
		for _, args := range rows {
			remoteExec, run := newLifecycleInvariantRoot(t, testsupport.HealthyRepositoryState())
			run(args)
			for _, command := range remoteExec.Commands {
				if forbiddenLifecycleScript.MatchString(command.Script) {
					t.Errorf("%q (%s) sent a runner-configuring script %s:\n%s", strings.Join(args, " "), name, command.ID, command.Script)
				}
				if strings.Contains(command.Script, "config.sh remove") {
					sentRemove = true
				}
			}
		}
	}
	// Guard against a vacuous pass: down legitimately removes the
	// registration, so the recording executor must have seen it.
	if !sentRemove {
		t.Fatalf("expected down/unregister to send config.sh remove through the recording executor")
	}
}
