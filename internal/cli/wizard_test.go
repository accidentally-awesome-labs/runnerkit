package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ux/stage"
)

func TestFirstRunWizardJSON(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	stateDir := t.TempDir()
	cmd := NewRootCommand(Dependencies{
		Version:        "test-version",
		In:             strings.NewReader(""),
		Out:            &out,
		Err:            &out,
		StateBaseDir:   stateDir,
		GitHub:         newFakePermittedGitHubService(),
		RemoteExecutor: newFakeRemoteExecutor(),
		Sleep:          noSleep,
		Prompts:        ui.NewCLIPrompter(strings.NewReader(""), &out),
		TTY:            ui.TerminalCapabilities{StdinTTY: false, StdoutTTY: true, Width: 80},
	})
	cmd.SetArgs([]string{"--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("json: %v\n%s", err, out.String())
	}
	if payload["command"] != "wizard" || payload["stage"] != string(stage.NoLocalState) {
		t.Fatalf("payload %#v", payload)
	}
}
