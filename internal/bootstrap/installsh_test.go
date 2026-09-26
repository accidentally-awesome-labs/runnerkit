package bootstrap

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const installShFixture = "#!/usr/bin/env bash\n" +
	"render_sudoers() {\n" +
	"\tlocal u=\"$1\"\n" +
	"\t" + InstallShSudoersBeginMarker + " (stale)\n" +
	"\tcat <<EOF\n" +
	"${u} ALL=(root) NOPASSWD: /usr/bin/apt-get\n" +
	"EOF\n" +
	"\t" + InstallShSudoersEndMarker + "\n" +
	"}\n" +
	"render_sudoers alice\n"

func TestReplaceInstallShSudoersBlockRewritesOnlyTheMarkedBlock(t *testing.T) {
	block, err := InstallShSudoersBlock()
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReplaceInstallShSudoersBlock(installShFixture)
	if err != nil {
		t.Fatal(err)
	}
	want := "#!/usr/bin/env bash\nrender_sudoers() {\n\tlocal u=\"$1\"\n" + block + "}\nrender_sudoers alice\n"
	if got != want {
		t.Fatalf("ReplaceInstallShSudoersBlock:\n--- got ---\n%s--- want ---\n%s", got, want)
	}
	again, err := ReplaceInstallShSudoersBlock(got)
	if err != nil || again != got {
		t.Fatalf("ReplaceInstallShSudoersBlock is not idempotent (err=%v)", err)
	}
	if !strings.Contains(block, "${u} ALL=(root) NOPASSWD: \\\\\n") {
		t.Fatalf("block does not substitute ${u} or escape the sudoers line continuation for the heredoc:\n%s", block)
	}
}

func TestReplaceInstallShSudoersBlockRejectsBadMarkers(t *testing.T) {
	begin := "\t" + InstallShSudoersBeginMarker + "\n"
	end := "\t" + InstallShSudoersEndMarker + "\n"
	for name, src := range map[string]string{
		"no markers":      "render_sudoers() {\n}\n",
		"begin only":      begin,
		"end only":        end,
		"end before":      end + begin,
		"duplicate begin": begin + begin + end,
		"duplicate end":   begin + end + end,
	} {
		if _, err := ReplaceInstallShSudoersBlock(src); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

// TestGenInstallShGeneratorRewritesStaleBlock runs the go:generate
// program itself (gen_installsh.go is `//go:build ignore`, so go build
// and go vet never compile it) against a stale copy of install.sh. CI
// runs `go test` but not `go generate`; this keeps the generator
// compiling and working.
func TestGenInstallShGeneratorRewritesStaleBlock(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles the generator with go run")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain not on PATH")
	}
	path := filepath.Join(t.TempDir(), "install.sh")
	if err := os.WriteFile(path, []byte(installShFixture), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(goBin, "run", "gen_installsh.go", "-path", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go run gen_installsh.go: %v\n%s", err, out)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want, err := ReplaceInstallShSudoersBlock(installShFixture)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("generator output differs from ReplaceInstallShSudoersBlock:\n%s", got)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("generator changed install.sh mode to %v", info.Mode().Perm())
	}
}
