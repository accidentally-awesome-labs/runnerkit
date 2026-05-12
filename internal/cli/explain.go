package cli

import (
	"fmt"
	"io"
	"strings"
)

func printExplainBlock(w io.Writer, step, why, runs, takes string) {
	if w == nil {
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Explain — %s\n", step)
	fmt.Fprintf(&b, "  WHY: %s\n", why)
	fmt.Fprintf(&b, "  RUNS: %s\n", runs)
	fmt.Fprintf(&b, "  TAKES: %s\n", takes)
	_, _ = fmt.Fprintln(w, b.String())
}

func explainInitDefault() (why, runs, takes string) {
	return "The runner host needs a one-time install so future runnerkit operations can use scoped sudo without interactive passwords.",
		"Shows install.sh URL and curl|sudo bash one-liner (or JSON next_actions).",
		"About one minute on a typical VM, mostly downloading the GitHub Actions runner bundle."
}

func explainBYOSetup() (why, runs, takes string) {
	return "RunnerKit will SSH as you, run preflight checks, then install or refresh the self-hosted runner using a short-lived registration token.",
		"Remote scripts under the runner install directory plus systemd unit management.",
		"Usually a few minutes depending on network and whether the runner binary is already cached."
}
