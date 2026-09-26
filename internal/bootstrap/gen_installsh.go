//go:build ignore

// gen_installsh rewrites the sudoers heredoc in install.sh (between the
// "# BEGIN runnerkit-sudoers" / "# END runnerkit-sudoers" markers) from
// bootstrap.RenderSudoersEntry. Run via `go generate ./internal/bootstrap`
// (or `go generate ./...`); CI checks `git diff --exit-code` afterwards.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/accidentally-awesome-labs/runnerkit/internal/bootstrap"
)

func main() {
	path := flag.String("path", "../../install.sh", "install.sh to rewrite in place")
	flag.Parse()
	src, err := os.ReadFile(*path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gen_installsh:", err)
		os.Exit(1)
	}
	out, err := bootstrap.ReplaceInstallShSudoersBlock(string(src))
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen_installsh: %s: %v\n", *path, err)
		os.Exit(1)
	}
	if out == string(src) {
		return
	}
	info, err := os.Stat(*path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gen_installsh:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*path, []byte(out), info.Mode().Perm()); err != nil {
		fmt.Fprintln(os.Stderr, "gen_installsh:", err)
		os.Exit(1)
	}
}
