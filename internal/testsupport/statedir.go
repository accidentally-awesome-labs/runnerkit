package testsupport

import (
	"fmt"
	"os"
)

// IsolateStateDir points RUNNERKIT_STATE_DIR (and XDG_STATE_HOME) at a fresh
// temporary directory for the whole test binary, so tests that do not inject
// Dependencies.StateBaseDir never touch the developer's real
// ~/.local/state/runnerkit nor the package directory (the v1.3.x stray
// internal/cli/sessions/ file). Call it from TestMain and defer the returned
// cleanup before os.Exit. It also sets RUNNERKIT_NO_UPDATE_NOTIFIER so the
// lazy update check never reaches api.github.com from a test.
func IsolateStateDir() (cleanup func(), err error) {
	dir, err := os.MkdirTemp("", "runnerkit-test-state-")
	if err != nil {
		return func() {}, fmt.Errorf("testsupport: create isolated state dir: %w", err)
	}
	prevState, hadState := os.LookupEnv("RUNNERKIT_STATE_DIR")
	prevXDG, hadXDG := os.LookupEnv("XDG_STATE_HOME")
	prevNotifier, hadNotifier := os.LookupEnv("RUNNERKIT_NO_UPDATE_NOTIFIER")
	_ = os.Setenv("RUNNERKIT_STATE_DIR", dir)
	_ = os.Setenv("XDG_STATE_HOME", dir)
	_ = os.Setenv("RUNNERKIT_NO_UPDATE_NOTIFIER", "1")
	return func() {
		restoreEnv("RUNNERKIT_STATE_DIR", prevState, hadState)
		restoreEnv("XDG_STATE_HOME", prevXDG, hadXDG)
		restoreEnv("RUNNERKIT_NO_UPDATE_NOTIFIER", prevNotifier, hadNotifier)
		_ = os.RemoveAll(dir)
	}, nil
}

// MainWithIsolatedStateDir is a TestMain body: it isolates the state dir,
// runs the tests and returns their exit code.
func MainWithIsolatedStateDir(run func() int) int {
	cleanup, err := IsolateStateDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer cleanup()
	return run()
}

func restoreEnv(key, value string, had bool) {
	if had {
		_ = os.Setenv(key, value)
		return
	}
	_ = os.Unsetenv(key)
}
