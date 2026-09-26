package bootstrap

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
)

// runnerRegistrationFloor is the oldest actions/runner release GitHub
// still accepts for new registrations (gaps.md
// runner-version-enforcement). The bundled pin must never fall below it.
const runnerRegistrationFloor = "2.329.0"

func parseRunnerVersion(t *testing.T, v string) [3]int {
	t.Helper()
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		t.Fatalf("runner version %q is not MAJOR.MINOR.PATCH", v)
	}
	var out [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			t.Fatalf("runner version %q: %v", v, err)
		}
		out[i] = n
	}
	return out
}

func TestRunnerPinAtOrAboveRegistrationFloor(t *testing.T) {
	pin := parseRunnerVersion(t, RunnerVersion)
	floor := parseRunnerVersion(t, runnerRegistrationFloor)
	for i := 0; i < 3; i++ {
		if pin[i] > floor[i] {
			break
		}
		if pin[i] < floor[i] {
			t.Fatalf("RunnerVersion %s is below the registration floor %s", RunnerVersion, runnerRegistrationFloor)
		}
	}
	for _, arch := range []string{"x64", "arm64"} {
		pkg, err := PackageFor("linux", arch)
		if err != nil {
			t.Fatalf("PackageFor linux/%s: %v", arch, err)
		}
		if pkg.Version != RunnerVersion ||
			!strings.Contains(pkg.Filename, RunnerVersion) ||
			!strings.Contains(pkg.URL, "/v"+RunnerVersion+"/") ||
			!strings.HasSuffix(pkg.URL, "/"+pkg.Filename) {
			t.Fatalf("linux/%s package does not match RunnerVersion %s: %#v", arch, RunnerVersion, pkg)
		}
		if len(pkg.SHA256) != 64 {
			t.Fatalf("linux/%s SHA256 %q is not a hex SHA-256", arch, pkg.SHA256)
		}
	}
}

// TestRenderedScriptsNeverDisableUpdate is the H-14 / N-15 invariant:
// RunnerKit pins a runner version but must never pass --disableupdate
// to config.sh. GitHub refuses jobs to runners that cannot self-update
// once their version falls out of support, so a disabled update turns
// an ageing pin into a silently dead runner.
func TestRenderedScriptsNeverDisableUpdate(t *testing.T) {
	pkg, err := PackageFor("linux", "x64")
	if err != nil {
		t.Fatal(err)
	}
	var scripts []struct{ name, body string }
	add := func(name, body string) { scripts = append(scripts, struct{ name, body string }{name, body}) }

	for _, mode := range []string{"", "ephemeral"} {
		for _, extra := range [][]string{nil, {"libsecret-1-dev", "dbus-x11"}} {
			for _, osID := range []string{"ubuntu", "fedora"} {
				opts := Options{
					RunnerName:    "runnerkit-owner-repo-local",
					RepoURL:       "https://github.com/owner/repo",
					Labels:        []string{"self-hosted", "runnerkit"},
					ServiceUser:   "runnerkit-runner",
					RunnerToken:   "tok",
					Package:       pkg,
					ExtraPackages: extra,
					OSReleaseID:   osID,
					Mode:          mode,
				}
				label := "mode=" + mode + " os=" + osID + " extra=" + strings.Join(extra, ",")
				rec := &recordingExecutor{}
				apply := Apply
				if mode == "ephemeral" {
					apply = ApplyEphemeral
				}
				if _, err := apply(context.Background(), rec, remote.Target{User: "alice", Host: "h", Port: 22}, opts); err != nil {
					t.Fatalf("%s: %v", label, err)
				}
				for _, c := range rec.commands {
					add(label+" "+c.ID, c.Script)
				}
				add(label+" RenderInstallScript", RenderInstallScript(opts))
				add(label+" RenderEphemeralInstallScript", RenderEphemeralInstallScript(opts))
				add(label+" RenderReconfigureScript", RenderReconfigureScript(opts))
			}
		}
	}
	add("RenderRemoveConfigScript", RenderRemoveConfigScript("/opt/actions-runner/runnerkit-x", ""))

	checkedConfigSh := 0
	for _, s := range scripts {
		if strings.Contains(strings.ToLower(s.body), "disableupdate") {
			t.Errorf("%s passes --disableupdate:\n%s", s.name, s.body)
		}
		if strings.Contains(s.body, "./config.sh --unattended") {
			checkedConfigSh++
		}
	}
	if checkedConfigSh == 0 {
		t.Fatal("no rendered script invoked ./config.sh --unattended; the invariant check is vacuous")
	}
}
