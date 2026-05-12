package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	rkstate "github.com/accidentally-awesome-labs/runnerkit/internal/state"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ui"
	"github.com/accidentally-awesome-labs/runnerkit/internal/ux/nextaction"
	"github.com/spf13/cobra"
)

type listOptions struct {
	host string
}

func newListCommand(deps Dependencies, jsonOutput *bool, noColor *bool) *cobra.Command {
	opts := &listOptions{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List RunnerKit-managed runners grouped by host",
		Long:  "Reads local state and prints each saved repository runner, optionally filtered to one SSH host (SEED-002).",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runList(deps, *jsonOutput, *noColor, opts)
		},
	}
	cmd.Flags().StringVar(&opts.host, "host", "", "filter to this SSH target (user@host or user@host:port; default SSH port 22)")
	return cmd
}

func runList(deps Dependencies, jsonOutput bool, noColor bool, opts *listOptions) error {
	renderer := newRenderer(deps, jsonOutput, noColor)
	ctx := context.Background()
	store := rkstate.NewStore(deps.StateBaseDir)
	repos, err := store.ListRepositories()
	if err != nil {
		_ = renderer.Error("state_io_failed", "RunnerKit can't read saved runner state.", []string{"Check permissions for " + store.Path() + "."})
		return NewExitError(ExitStateIO, err)
	}

	var filterKey string
	if strings.TrimSpace(opts.host) != "" {
		k, err := remote.CanonicalHostKey(opts.host, 22)
		if err != nil {
			_ = renderer.Error("invalid_host", "Invalid --host value.", []string{err.Error()})
			return NewExitError(ExitInvalidInput, err)
		}
		filterKey = k
	}

	type hostBucket struct {
		key   string
		items []listItemJSON
	}
	buckets := map[string]*hostBucket{}
	order := []string{}

	for _, rs := range repos {
		if strings.TrimSpace(rs.Machine.HostRef) == "" {
			continue
		}
		port := rs.Machine.Port
		if port == 0 {
			port = 22
		}
		key, err := remote.CanonicalHostKey(rs.Machine.HostRef, port)
		if err != nil {
			continue
		}
		if filterKey != "" && key != filterKey {
			continue
		}
		b, ok := buckets[key]
		if !ok {
			b = &hostBucket{key: key}
			buckets[key] = b
			order = append(order, key)
		}
		st := collectStatus(ctx, deps, store.Path(), rs, true)
		b.items = append(b.items, listItemFromResult(rs, st))
	}

	sort.Strings(order)

	if jsonOutput {
		hosts := make([]map[string]any, 0, len(order))
		for _, k := range order {
			b := buckets[k]
			rows := make([]any, len(b.items))
			for i := range b.items {
				rows[i] = b.items[i].toMap()
			}
			hosts = append(hosts, map[string]any{
				"host_ref": b.key,
				"repos":    rows,
			})
		}
		payload := map[string]any{
			"ok":         true,
			"command":    "list",
			"state_path": store.Path(),
			"hosts":      hosts,
		}
		nextaction.MergePayload(payload, "", nil)
		return renderer.JSON(payload)
	}

	if len(order) == 0 {
		msg := "No RunnerKit-managed runners in local state."
		if filterKey != "" {
			msg = "No saved runners for host " + filterKey + "."
		}
		return renderer.Step(1, 1, "runner inventory", ui.WarningLine(msg), ui.Bullet("Run `runnerkit up --repo owner/name --host user@host` or `runnerkit register` after `runnerkit init`."))
	}

	lines := []ui.Line{}
	for _, k := range order {
		b := buckets[k]
		lines = append(lines, ui.Bullet("Host "+b.key))
		for _, it := range b.items {
			lines = append(lines, ui.Bullet(fmt.Sprintf("%s — %s (%s)", it.Repo, it.RunnerName, it.HealthSummary)))
		}
	}
	return renderer.Step(1, 1, "runner inventory", lines...)
}

type listItemJSON struct {
	Repo          string `json:"repo"`
	RunnerName    string `json:"runner_name"`
	Mode          string `json:"mode"`
	InstallPath   string `json:"install_path,omitempty"`
	ServiceName   string `json:"service_name,omitempty"`
	HealthSummary string `json:"health_summary"`
}

func listItemFromResult(rs rkstate.RepositoryState, st statusResult) listItemJSON {
	h := string(st.Health.State)
	if strings.TrimSpace(st.Health.Summary) != "" {
		h = st.Health.Summary
	}
	if h == "" {
		h = "unknown"
	}
	return listItemJSON{
		Repo:          rs.Repo.FullName,
		RunnerName:    rs.Runner.Name,
		Mode:          rs.Runner.Mode,
		InstallPath:   rs.Machine.InstallPath,
		ServiceName:   rs.Machine.ServiceName,
		HealthSummary: h,
	}
}

func (it listItemJSON) toMap() map[string]any {
	m := map[string]any{
		"repo":           it.Repo,
		"runner_name":    it.RunnerName,
		"mode":           it.Mode,
		"health_summary": it.HealthSummary,
	}
	if strings.TrimSpace(it.InstallPath) != "" {
		m["install_path"] = it.InstallPath
	}
	if strings.TrimSpace(it.ServiceName) != "" {
		m["service_name"] = it.ServiceName
	}
	return m
}
