package cli

import (
	"bytes"
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	gh "github.com/accidentally-awesome-labs/runnerkit/internal/github"
	"github.com/accidentally-awesome-labs/runnerkit/internal/provider"
	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/accidentally-awesome-labs/runnerkit/internal/testsupport"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
)

// guardHarness wires recording fakes for the v1.3.4 harm-reduction guard
// tests (backlog A-04, A-05, A-08, A-15). Every refusal must happen
// before any provider, SSH, or (where stated) GitHub call.
type guardHarness struct {
	stateDir string
	github   *testsupport.GitHubService
	remote   *fakeRemoteExecutor
	cloud    *provider.FakeProvider
	out      bytes.Buffer
	errOut   bytes.Buffer
}

func newGuardHarness(t *testing.T) *guardHarness {
	t.Helper()
	return &guardHarness{
		stateDir: t.TempDir(),
		github:   &testsupport.GitHubService{Repo: gh.Repo{Host: "github.com", Owner: "owner", Name: "repo", FullName: testsupport.TestRepoFullName, Private: true}},
		remote:   newFakeRemoteExecutor(),
		cloud:    &provider.FakeProvider{},
	}
}

func (h *guardHarness) run(t *testing.T, prompts ui.Prompter, tty bool, args ...string) error {
	t.Helper()
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		Out:            &h.out,
		Err:            &h.errOut,
		StateBaseDir:   h.stateDir,
		GitHub:         h.github,
		RemoteExecutor: h.remote,
		Providers:      provider.NewRegistry(h.cloud),
		CommandRunner:  staticCommandRunner{remote: "git@github.com:owner/repo.git"},
		Prompts:        prompts,
		TTY:            ui.TerminalCapabilities{StdinTTY: tty, StdoutTTY: tty, Width: 80},
		Sleep:          noSleep,
		Clock:          func() time.Time { return time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC) },
	})
	cmd.SetArgs(args)
	return cmd.Execute()
}

func (h *guardHarness) combined() string {
	return strings.Join(strings.Fields(h.out.String()+h.errOut.String()), " ")
}

func (h *guardHarness) seed(t *testing.T, repo state.RepositoryState) []byte {
	t.Helper()
	store := state.NewStore(h.stateDir)
	if err := store.Save(testsupport.StateWithRepository(repo)); err != nil {
		t.Fatalf("seed state: %v", err)
	}
	before, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatalf("read seeded state: %v", err)
	}
	return before
}

func (h *guardHarness) assertStateUnchanged(t *testing.T, before []byte) {
	t.Helper()
	after, err := os.ReadFile(state.NewStore(h.stateDir).Path())
	if err != nil {
		t.Fatalf("read state after command: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("state.json changed:\nbefore=%s\nafter=%s", before, after)
	}
}

func (h *guardHarness) assertNoProviderOrSSH(t *testing.T) {
	t.Helper()
	c := h.cloud
	if c.ValidateCalls != 0 || c.PlanCalls != 0 || c.ProvisionCalls != 0 || c.WaitReadyCalls != 0 || c.DescribeCalls != 0 || c.DestroyCalls != 0 || c.VerifyDestroyedCalls != 0 {
		t.Fatalf("provider was called: %#v", c)
	}
	if h.remote.probeCalls != 0 || len(h.remote.runs) != 0 {
		t.Fatalf("SSH executor was called: probe=%d runs=%d", h.remote.probeCalls, len(h.remote.runs))
	}
}

func (h *guardHarness) assertNoGitHub(t *testing.T) {
	t.Helper()
	g := h.github
	if g.RepositoryCalls != 0 || g.VerifyAuthCalls != 0 || g.VerifyRunnerManagementReadCalls != 0 || g.CreateRegistrationTokenCalls != 0 || g.CreateRemovalTokenCalls != 0 || g.ListRunnersCalls != 0 || g.DeleteRunnerCalls != 0 {
		t.Fatalf("GitHub was called: %#v", g)
	}
}

func assertExitInvalidInput(t *testing.T, h *guardHarness, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s refusal, got nil error\n%s", code, h.combined())
	}
	if got := ExitCode(err); got != ExitInvalidInput {
		t.Fatalf("ExitCode() = %d, want %d (err=%v)\n%s", got, ExitInvalidInput, err, h.combined())
	}
	// JSON output carries the code in the payload; human output carries it
	// in the returned error.
	if !strings.Contains(h.combined(), code) && !strings.Contains(err.Error(), code) {
		t.Fatalf("error code %q missing from output and error %q:\n%s", code, err.Error(), h.combined())
	}
}

// --- A-04: down/unregister refuse cloud state -------------------------

func testDownRefusesCloudState(t *testing.T, verb string, extra ...string) {
	t.Helper()
	for _, fixture := range []struct {
		name string
		repo state.RepositoryState
	}{
		{"persistent", testsupport.CloudRepositoryState()},
		{"ephemeral", testsupport.EphemeralCloudRepositoryState()},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			h := newGuardHarness(t)
			before := h.seed(t, fixture.repo)
			args := append([]string{"--json", verb, "--repo", testsupport.TestRepoFullName}, extra...)
			err := h.run(t, nil, false, append(args, "--no-color")...)
			assertExitInvalidInput(t, h, err, "wrong_cleanup_command")
			if !strings.Contains(h.combined(), "runnerkit destroy --repo owner/repo") {
				t.Fatalf("refusal must point at destroy:\n%s", h.combined())
			}
			if strings.Contains(h.combined(), `"state_removed":true`) {
				t.Fatalf("down reported state removal on cloud state:\n%s", h.combined())
			}
			h.assertStateUnchanged(t, before)
			h.assertNoProviderOrSSH(t)
			h.assertNoGitHub(t)
		})
	}
}

func TestDown_RefusesCloudState(t *testing.T) {
	testDownRefusesCloudState(t, "down", "--yes")
}

func TestDown_DryRunRefusesCloudState(t *testing.T) {
	testDownRefusesCloudState(t, "down", "--dry-run")
}

func TestUnregisterAlias_RefusesCloudState(t *testing.T) {
	testDownRefusesCloudState(t, "unregister", "--yes")
}

// --- A-05: up --replace refuses live cloud state ----------------------

type replacePhrasePrompter struct{ inputs []string }

func (p *replacePhrasePrompter) Confirm(context.Context, ui.Prompt) (bool, error) { return true, nil }
func (p *replacePhrasePrompter) Select(_ context.Context, _ ui.Prompt, options []ui.Option) (string, error) {
	return options[0].Value, nil
}
func (p *replacePhrasePrompter) Input(_ context.Context, prompt ui.Prompt) (string, error) {
	p.inputs = append(p.inputs, prompt.Message)
	return "replace owner/repo", nil
}

func TestUpCloudReplace_RefusesLiveCloudState(t *testing.T) {
	cases := map[string][]string{
		"yes_replace": {"up", "--repo", "owner/repo", "--experimental", "--cloud", "hetzner", "--cloud-region", "nbg1", "--yes", "--replace", "--no-color"},
		"dry_run":     {"up", "--repo", "owner/repo", "--experimental", "--cloud", "hetzner", "--cloud-region", "nbg1", "--yes", "--dry-run", "--no-color"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			h := newGuardHarness(t)
			before := h.seed(t, testsupport.CloudRepositoryState())
			err := h.run(t, nil, false, args...)
			assertExitInvalidInput(t, h, err, "cloud_state_exists")
			for _, want := range []string{"server:srv-123", "firewall:fw-123", "ssh_key:key-123", "runnerkit destroy --repo owner/repo"} {
				if !strings.Contains(h.combined(), want) {
					t.Fatalf("refusal missing %q:\n%s", want, h.combined())
				}
			}
			if h.cloud.ProvisionCalls != 0 {
				t.Fatalf("Provision called %d times", h.cloud.ProvisionCalls)
			}
			h.assertStateUnchanged(t, before)
			h.assertNoProviderOrSSH(t)
		})
	}
}

func TestUpCloudReplace_InteractivePhraseRefusesLiveCloudState(t *testing.T) {
	h := newGuardHarness(t)
	before := h.seed(t, testsupport.CloudRepositoryState())
	prompts := &replacePhrasePrompter{}
	err := h.run(t, prompts, true, "up", "--repo", "owner/repo", "--experimental", "--cloud", "hetzner", "--cloud-region", "nbg1", "--no-color")
	assertExitInvalidInput(t, h, err, "cloud_state_exists")
	h.assertStateUnchanged(t, before)
	h.assertNoProviderOrSSH(t)
}

func TestUpBYOReplace_RefusesLiveCloudState(t *testing.T) {
	h := newGuardHarness(t)
	before := h.seed(t, testsupport.CloudRepositoryState())
	err := h.run(t, nil, false, "up", "--repo", "owner/repo", "--host", "alice@example.com", "--yes", "--replace", "--no-color")
	assertExitInvalidInput(t, h, err, "cloud_state_exists")
	h.assertStateUnchanged(t, before)
	h.assertNoProviderOrSSH(t)
	if h.github.CreateRegistrationTokenCalls != 0 {
		t.Fatalf("registration token minted before refusal")
	}
}

func TestRefuseIfLiveCloudState_AllowsBYOAndEmptyCloudState(t *testing.T) {
	renderer := newRenderer(Dependencies{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}}, false, true)
	if err := refuseIfLiveCloudState(renderer, testsupport.HealthyRepositoryState(), true, "owner/repo"); err != nil {
		t.Fatalf("BYO state must not be refused: %v", err)
	}
	empty := testsupport.CloudRepositoryState()
	empty.Provider.IDs = nil
	empty.Provider.ResourceIDs = nil
	empty.Provider.Cloud = state.CloudInventory{Provider: "hetzner"}
	empty.Cleanup.ProviderResourceIDs = []string{}
	if err := refuseIfLiveCloudState(renderer, empty, true, "owner/repo"); err != nil {
		t.Fatalf("cloud state without resource IDs must not be refused: %v", err)
	}
	if err := refuseIfLiveCloudState(renderer, testsupport.CloudRepositoryState(), false, "owner/repo"); err != nil {
		t.Fatalf("missing state must not be refused: %v", err)
	}
	onlyInventory := testsupport.CloudRepositoryState()
	onlyInventory.Provider.IDs = nil
	onlyInventory.Provider.ResourceIDs = nil
	onlyInventory.Cleanup.ProviderResourceIDs = []string{}
	if err := refuseIfLiveCloudState(renderer, onlyInventory, true, "owner/repo"); err == nil {
		t.Fatal("cloud inventory server/firewall/ssh-key IDs must be refused")
	}
	onlyCleanup := testsupport.CloudRepositoryState()
	onlyCleanup.Provider.IDs = nil
	onlyCleanup.Provider.ResourceIDs = nil
	onlyCleanup.Provider.Cloud = state.CloudInventory{Provider: "hetzner"}
	if got := liveCloudResourceIDs(onlyCleanup); len(got) == 0 || got[0] != "server:srv-123" {
		t.Fatalf("Cleanup.ProviderResourceIDs must count as live cloud IDs, got %v", got)
	}
	if err := refuseIfLiveCloudState(renderer, onlyCleanup, true, "owner/repo"); err == nil {
		t.Fatal("cleanup provider_resource_ids must be refused")
	}
	nonBillable := testsupport.CloudRepositoryState()
	nonBillable.Provider.IDs = map[string]string{"create_action": "act-1"}
	nonBillable.Provider.ResourceIDs = nil
	nonBillable.Provider.Cloud = state.CloudInventory{Provider: "hetzner"}
	nonBillable.Cleanup.ProviderResourceIDs = []string{"create_action:act-1"}
	if err := refuseIfLiveCloudState(renderer, nonBillable, true, "owner/repo"); err != nil {
		t.Fatalf("non-billable action IDs must not be refused: %v", err)
	}
}

// --- A-08: ephemeral cloud disabled ------------------------------------

func TestUp_EphemeralCloudDisabled(t *testing.T) {
	cases := map[string][]string{
		"no_experimental":   {"up", "--repo", "owner/repo", "--mode", "ephemeral", "--cloud", "hetzner", "--dry-run", "--no-color"},
		"with_experimental": {"up", "--repo", "owner/repo", "--mode", "ephemeral", "--cloud", "hetzner", "--experimental", "--cloud-region", "nbg1", "--yes", "--dry-run", "--no-color"},
		"json":              {"--json", "up", "--repo", "owner/repo", "--mode", "ephemeral", "--cloud", "hetzner", "--experimental", "--cloud-region", "nbg1", "--yes", "--no-color"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			h := newGuardHarness(t)
			err := h.run(t, nil, false, args...)
			assertExitInvalidInput(t, h, err, "ephemeral_cloud_disabled")
			if !strings.Contains(h.combined(), "GitHub-hosted runners") {
				t.Fatalf("refusal must recommend GitHub-hosted runners:\n%s", h.combined())
			}
			if h.cloud.ValidateCalls != 0 {
				t.Fatalf("Provider.Validate called %d times", h.cloud.ValidateCalls)
			}
			h.assertNoProviderOrSSH(t)
			h.assertNoGitHub(t)
		})
	}
}

// interactiveChoicePrompter answers Select prompts from a fixed list.
type interactiveChoicePrompter struct{ answers []string }

func (p *interactiveChoicePrompter) Confirm(context.Context, ui.Prompt) (bool, error) {
	return true, nil
}
func (p *interactiveChoicePrompter) Select(_ context.Context, _ ui.Prompt, options []ui.Option) (string, error) {
	if len(p.answers) == 0 {
		return options[0].Value, nil
	}
	next := p.answers[0]
	p.answers = p.answers[1:]
	return next, nil
}
func (p *interactiveChoicePrompter) Input(context.Context, ui.Prompt) (string, error) {
	return "alice@example.com", nil
}

func TestUp_InteractiveChoicesStillGated(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		answers []string
		code    string
	}{
		{"cloud_ephemeral", []string{"up", "--repo", "owner/repo", "--experimental", "--dry-run", "--no-color"}, []string{"cloud", "ephemeral"}, "ephemeral_cloud_disabled"},
		{"cloud_persistent", []string{"up", "--repo", "owner/repo", "--dry-run", "--no-color"}, []string{"cloud", "persistent"}, "experimental_required"},
		{"byo_ephemeral", []string{"up", "--repo", "owner/repo", "--dry-run", "--no-color"}, []string{"byo", "ephemeral"}, "experimental_required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newGuardHarness(t)
			err := h.run(t, &interactiveChoicePrompter{answers: tc.answers}, true, tc.args...)
			assertExitInvalidInput(t, h, err, tc.code)
			h.assertNoProviderOrSSH(t)
		})
	}
}

func TestRegister_InteractiveCloudChoiceRefused(t *testing.T) {
	// register has no --cloud / --cloud-region flags, so an interactive
	// Cloud pick must not end in an experimental/region refusal whose
	// advice register cannot follow.
	for _, extra := range [][]string{nil, {"--experimental"}} {
		h := newGuardHarness(t)
		args := append([]string{"register", "--repo", "owner/repo", "--dry-run", "--no-color"}, extra...)
		err := h.run(t, &interactiveChoicePrompter{answers: []string{"cloud", "persistent"}}, true, args...)
		if err == nil || ExitCode(err) != ExitInvalidInput {
			t.Fatalf("expected invalid_register_cloud exit %d, got %v\n%s", ExitInvalidInput, err, h.combined())
		}
		if !strings.Contains(h.combined(), "runnerkit register is BYO-only") || strings.Contains(h.combined(), "experimental_required") || strings.Contains(h.combined(), "cloud_region_required") {
			t.Fatalf("expected the invalid_register_cloud refusal:\n%s", h.combined())
		}
		if !strings.Contains(h.combined(), "runnerkit up --repo ... --experimental --cloud hetzner --cloud-region <location>") {
			t.Fatalf("refusal must point at up --experimental --cloud:\n%s", h.combined())
		}
		h.assertNoProviderOrSSH(t)
	}
}

// --- A-15: --experimental and explicit --cloud-region -----------------

func TestUp_CloudRequiresExperimental(t *testing.T) {
	for _, extra := range [][]string{{"--dry-run"}, {"--yes"}, {"--cloud-region", "nbg1", "--yes", "--dry-run"}} {
		h := newGuardHarness(t)
		args := append([]string{"up", "--repo", "owner/repo", "--cloud", "hetzner", "--no-color"}, extra...)
		err := h.run(t, nil, false, args...)
		assertExitInvalidInput(t, h, err, "experimental_required")
		if !strings.Contains(h.combined(), "billed by Hetzner") {
			t.Fatalf("refusal must say why:\n%s", h.combined())
		}
		h.assertNoProviderOrSSH(t)
		h.assertNoGitHub(t)
	}
}

func TestUp_CloudRequiresRegion(t *testing.T) {
	h := newGuardHarness(t)
	err := h.run(t, nil, false, "up", "--repo", "owner/repo", "--cloud", "hetzner", "--experimental", "--yes", "--dry-run", "--no-color")
	assertExitInvalidInput(t, h, err, "cloud_region_required")
	if strings.Contains(h.combined(), "fsn1") {
		t.Fatalf("refusal must not suggest a hard-coded location:\n%s", h.combined())
	}
	h.assertNoProviderOrSSH(t)
	h.assertNoGitHub(t)
}

func TestUp_CloudWithExperimentalAndRegionReachesProvider(t *testing.T) {
	h := newGuardHarness(t)
	if err := h.run(t, nil, false, "up", "--repo", "owner/repo", "--cloud", "hetzner", "--experimental", "--cloud-region", "nbg1", "--yes", "--dry-run", "--no-color"); err != nil {
		t.Fatalf("experimental cloud dry-run returned error: %v\n%s", err, h.combined())
	}
	if h.cloud.ValidateCalls != 1 || h.cloud.PlanCalls != 1 || h.cloud.ProvisionCalls != 0 {
		t.Fatalf("unexpected provider calls: %#v", h.cloud)
	}
	if len(h.cloud.ValidateInputs) != 1 || h.cloud.ValidateInputs[0].Profile.Region != "nbg1" {
		t.Fatalf("explicit --cloud-region not passed to provider: %#v", h.cloud.ValidateInputs)
	}
}

func TestUp_EphemeralBYORequiresExperimental(t *testing.T) {
	for _, verb := range []string{"up", "register"} {
		t.Run(verb, func(t *testing.T) {
			h := newGuardHarness(t)
			err := h.run(t, nil, false, verb, "--repo", "owner/repo", "--host", "alice@example.com", "--mode", "ephemeral", "--yes", "--no-color")
			assertExitInvalidInput(t, h, err, "experimental_required")
			if !strings.Contains(h.combined(), "not isolation") {
				t.Fatalf("refusal must say BYO ephemeral is not isolation:\n%s", h.combined())
			}
			h.assertNoProviderOrSSH(t)
			h.assertNoGitHub(t)
		})
	}
}

func TestUp_BYOPersistentDoesNotRequireExperimental(t *testing.T) {
	h := newGuardHarness(t)
	if err := h.run(t, nil, false, "up", "--repo", "owner/repo", "--host", "alice@example.com", "--yes", "--dry-run", "--no-color"); err != nil {
		t.Fatalf("BYO persistent dry-run returned error: %v\n%s", err, h.combined())
	}
	h2 := newGuardHarness(t)
	if err := h2.run(t, nil, false, "up", "--repo", "owner/repo", "--host", "alice@example.com", "--mode", "ephemeral", "--experimental", "--yes", "--dry-run", "--no-color"); err != nil {
		t.Fatalf("BYO ephemeral with --experimental returned error: %v\n%s", err, h2.combined())
	}
	if !strings.Contains(h2.combined(), "experimental and not isolation") {
		t.Fatalf("BYO ephemeral output must label it experimental and not isolation:\n%s", h2.combined())
	}
}

func TestUpAndRegister_CloudRegionHasNoDefault(t *testing.T) {
	deps := Dependencies{Version: "test-version", Out: &bytes.Buffer{}, Err: &bytes.Buffer{}}
	jsonOut, noColor := false, false
	up := newUpCommand(deps, &jsonOut, &noColor)
	if f := up.Flags().Lookup("cloud-region"); f == nil || f.DefValue != "" {
		t.Fatalf("up --cloud-region must have no default, got %#v", f)
	}
	for _, cmd := range []string{"up", "register"} {
		c := newUpCommand(deps, &jsonOut, &noColor)
		if cmd == "register" {
			c = newRegisterCommand(deps, &jsonOut, &noColor)
		}
		if c.Flags().Lookup("experimental") == nil {
			t.Fatalf("%s is missing --experimental", cmd)
		}
	}
}

// --- A-08 / A-09 copy contracts ----------------------------------------

// nonTestGoStringLiterals returns every string literal (decoded) in
// non-test Go files under the repository's cmd/ and internal/ trees.
func nonTestGoStringLiterals(t *testing.T) map[string][]string {
	t.Helper()
	out := map[string][]string{}
	for _, root := range []string{"../../cmd", "../../internal"} {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			ast.Inspect(file, func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				value, err := strconv.Unquote(lit.Value)
				if err != nil {
					value = lit.Value
				}
				out[path] = append(out[path], value)
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
	return out
}

func TestNoCopyRecommendsEphemeralCloud(t *testing.T) {
	// Go half of the A-08 contract; the docs/**/*.md half lands with the
	// v1.3.4 docs pass.
	for path, literals := range nonTestGoStringLiterals(t) {
		for _, lit := range literals {
			if strings.Contains(lit, "--mode ephemeral --cloud") {
				t.Errorf("%s: string literal recommends ephemeral cloud: %q", path, lit)
			}
		}
	}
}

// Cloud `up` is refused without --experimental (experimental_required) and
// --cloud-region, so copy that tells users to re-run cloud up must carry
// both. Concatenated fragments starting with " --cloud hetzner" are the
// tail of such a command. Descriptive strings (e.g. "--cloud hetzner needs
// an explicit --cloud-region") do not match either pattern.
func TestNoCopyRecommendsCloudUpWithoutExperimental(t *testing.T) {
	for path, literals := range nonTestGoStringLiterals(t) {
		for _, lit := range literals {
			if !strings.Contains(lit, "--cloud hetzner") || strings.Contains(lit, "--experimental") {
				continue
			}
			if strings.Contains(lit, "runnerkit up") || strings.HasPrefix(lit, " --cloud hetzner") {
				t.Errorf("%s: string literal recommends cloud up without --experimental --cloud-region: %q", path, lit)
			}
		}
	}
}

func TestNoMisleadingRecommendationCopy(t *testing.T) {
	banned := []string{"recommended default", "recommended cloud", "NOT a blanket"}
	for path, literals := range nonTestGoStringLiterals(t) {
		for _, lit := range literals {
			for _, phrase := range banned {
				if strings.Contains(lit, phrase) {
					t.Errorf("%s: string literal contains %q: %q", path, phrase, lit)
				}
			}
		}
	}
	// The installer sudoers fragment is root-equivalent; its comment must
	// not claim otherwise.
	sudoers, err := os.ReadFile("../../internal/bootstrap/sudoers.go")
	if err != nil {
		t.Fatalf("read sudoers.go: %v", err)
	}
	if strings.Contains(string(sudoers), "NOT a blanket") {
		t.Error("internal/bootstrap/sudoers.go still claims the installer sudoers is NOT a blanket NOPASSWD ALL")
	}
	if !strings.Contains(string(sudoers), "Root-equivalent") {
		t.Error("internal/bootstrap/sudoers.go must say the installer sudoers is root-equivalent")
	}
	goreleaser, err := os.ReadFile("../../.goreleaser.yaml")
	if err != nil {
		t.Fatalf("read .goreleaser.yaml: %v", err)
	}
	for _, phrase := range append(banned, "Reliable GitHub Actions self-hosted runners") {
		if strings.Contains(string(goreleaser), phrase) {
			t.Errorf(".goreleaser.yaml contains %q", phrase)
		}
	}
	if !strings.Contains(string(goreleaser), "description: CLI to set up and check GitHub Actions self-hosted runners (experimental)") {
		t.Error(".goreleaser.yaml homebrew cask description not updated")
	}
}
