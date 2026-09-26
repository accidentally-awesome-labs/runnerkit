package cli

import (
	"os"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/testsupport"
)

// TestMain keeps the package hermetic (A-03 / C-02): normalizeDependencies
// now defaults StateBaseDir to the real state directory, so tests that do
// not inject one must resolve it to a throwaway directory instead of the
// developer's ~/.local/state/runnerkit.
func TestMain(m *testing.M) {
	os.Exit(testsupport.MainWithIsolatedStateDir(m.Run))
}
