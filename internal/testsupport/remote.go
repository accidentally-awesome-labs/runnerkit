package testsupport

import (
	"context"

	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
)

type RemoteExecutor struct {
	ProbeResult        remote.ProbeResult
	ProbeErr           error
	Commands           []remote.Command
	Results            map[string]remote.Result
	Errors             map[string]error
	ProbeHostKeyResult remote.HostKey
	ProbeHostKeyErr    error
	ProbeHostKeyCalls  int
}

func (r *RemoteExecutor) Probe(context.Context, remote.Target) (remote.ProbeResult, error) {
	if r.ProbeResult.Commands == nil {
		r.ProbeResult.Commands = map[string]bool{}
	}
	return r.ProbeResult, r.ProbeErr
}

func (r *RemoteExecutor) Run(_ context.Context, _ remote.Target, command remote.Command) (remote.Result, error) {
	r.Commands = append(r.Commands, command)
	if r.Results != nil {
		if result, ok := r.Results[command.ID]; ok {
			return result, r.Errors[command.ID]
		}
	}
	return remote.Result{Stdout: "ok", ExitCode: 0}, nil
}

func (r *RemoteExecutor) ProbeHostKey(context.Context, remote.Target) (remote.HostKey, error) {
	r.ProbeHostKeyCalls++
	return r.ProbeHostKeyResult, r.ProbeHostKeyErr
}

func (r *RemoteExecutor) CommandIDs() []string {
	ids := make([]string, 0, len(r.Commands))
	for _, command := range r.Commands {
		ids = append(ids, command.ID)
	}
	return ids
}
