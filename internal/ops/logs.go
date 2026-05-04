package ops

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/accidentally-awesome-labs/runnerkit/internal/remote"
	"github.com/accidentally-awesome-labs/runnerkit/internal/state"
)

const (
	CommandLogsSystemdJournal       = "logs.systemd.journal"
	CommandLogsRunnerDiagList       = "logs.runner.diag.list"
	CommandLogsRunnerDiagTail       = "logs.runner.diag.tail"
	CommandLogsEphemeralArchiveList = "logs.ephemeral.archive.list"
	CommandLogsEphemeralArchiveTail = "logs.ephemeral.archive.tail"
)

type LogSection struct {
	Source   string   `json:"source"`
	Title    string   `json:"title"`
	Metadata string   `json:"metadata"`
	Content  string   `json:"content"`
	Warnings []string `json:"warnings"`
}

type LogBundle struct {
	Repo      string       `json:"repo"`
	StatePath string       `json:"state_path"`
	Since     string       `json:"since"`
	Lines     int          `json:"lines"`
	Sections  []LogSection `json:"sections"`
	Warnings  []string     `json:"warnings"`
}

func CollectLogs(ctx context.Context, executor remote.Executor, target remote.Target, repoState state.RepositoryState, since string, lines int) LogBundle {
	if executor == nil {
		executor = remote.UnavailableExecutor{}
	}
	if strings.TrimSpace(since) == "" {
		since = "1h"
	}
	if lines < 1 {
		lines = 1
	}
	if lines > 1000 {
		lines = 1000
	}
	bundle := LogBundle{Repo: repoState.Repo.FullName, Since: since, Lines: lines, Sections: []LogSection{}, Warnings: []string{}}
	journalScript := "journalctl -u " + shellQuote(repoState.Machine.ServiceName) + " --since " + shellQuote(since) + " -n " + strconv.Itoa(lines) + " --no-pager"
	journal, err := executor.Run(ctx, target, remote.Command{ID: CommandLogsSystemdJournal, Script: journalScript, Timeout: 15 * time.Second})
	if err != nil || journal.ExitCode != 0 {
		bundle.Warnings = append(bundle.Warnings, "systemd journal unavailable")
	} else {
		bundle.Sections = append(bundle.Sections, LogSection{Source: "systemd", Title: "systemd journal", Metadata: repoState.Machine.ServiceName, Content: journal.Stdout})
	}

	diagListScript := "find " + shellQuote(repoState.Machine.InstallPath+"/_diag") + " -maxdepth 1 -type f \\( -name 'Runner_*.log' -o -name 'Worker_*.log' \\) -printf '%T@ %p\\n' 2>/dev/null | sort -nr | head -n 4 | cut -d' ' -f2-"
	diagList, err := executor.Run(ctx, target, remote.Command{ID: CommandLogsRunnerDiagList, Script: diagListScript, Timeout: 15 * time.Second})
	if err != nil || diagList.ExitCode != 0 {
		bundle.Warnings = append(bundle.Warnings, "runner diag list unavailable")
	} else {
		paths := nonEmptyLines(diagList.Stdout)
		if len(paths) == 0 {
			bundle.Sections = append(bundle.Sections, LogSection{Source: "runner_diag", Title: "runner diag", Metadata: repoState.Machine.InstallPath + "/_diag", Content: "No runner diag files found."})
		} else {
			quoted := make([]string, 0, len(paths))
			for _, path := range paths {
				quoted = append(quoted, shellQuote(path))
			}
			tailScript := "tail -n " + strconv.Itoa(lines) + " " + strings.Join(quoted, " ")
			tail, tailErr := executor.Run(ctx, target, remote.Command{ID: CommandLogsRunnerDiagTail, Script: tailScript, Timeout: 15 * time.Second})
			if tailErr != nil || tail.ExitCode != 0 {
				bundle.Warnings = append(bundle.Warnings, "runner diag tail unavailable")
			} else {
				bundle.Sections = append(bundle.Sections, LogSection{Source: "runner_diag", Title: "runner diag", Metadata: strings.Join(paths, ", "), Content: tail.Stdout})
			}
		}
	}
	// Ephemeral runners also expose preserved archive logs that the
	// finalizer/cleanup writes outside the install path. We collect a
	// bounded set of those files so `runnerkit logs` can show preserved
	// _diag and journal output even after `down`/`destroy` removes the
	// runner files.
	if repoState.Runner.Mode == "ephemeral" && strings.TrimSpace(repoState.Ephemeral.LogArchivePath) != "" {
		archive := repoState.Ephemeral.LogArchivePath
		listScript := "find " + shellQuote(archive) + " -maxdepth 1 -type f \\( -name 'Runner_*.log' -o -name 'Worker_*.log' -o -name 'systemd-journal.log' \\) 2>/dev/null"
		archiveList, err := executor.Run(ctx, target, remote.Command{ID: CommandLogsEphemeralArchiveList, Script: listScript, Timeout: 15 * time.Second})
		if err != nil || archiveList.ExitCode != 0 {
			bundle.Warnings = append(bundle.Warnings, "ephemeral archive list unavailable")
		} else {
			paths := nonEmptyLines(archiveList.Stdout)
			if len(paths) > 0 {
				quoted := make([]string, 0, len(paths))
				diagPaths := []string{}
				journalPaths := []string{}
				for _, path := range paths {
					quoted = append(quoted, shellQuote(path))
					if strings.HasSuffix(path, "systemd-journal.log") {
						journalPaths = append(journalPaths, path)
					} else {
						diagPaths = append(diagPaths, path)
					}
				}
				tailScript := "tail -n " + strconv.Itoa(lines) + " " + strings.Join(quoted, " ")
				tail, tailErr := executor.Run(ctx, target, remote.Command{ID: CommandLogsEphemeralArchiveTail, Script: tailScript, Timeout: 15 * time.Second})
				if tailErr != nil || tail.ExitCode != 0 {
					bundle.Warnings = append(bundle.Warnings, "ephemeral archive tail unavailable")
				} else {
					if len(diagPaths) > 0 {
						bundle.Sections = append(bundle.Sections, LogSection{Source: "ephemeral_runner_diag", Title: "ephemeral archive _diag", Metadata: strings.Join(diagPaths, ", "), Content: tail.Stdout})
					}
					if len(journalPaths) > 0 {
						bundle.Sections = append(bundle.Sections, LogSection{Source: "ephemeral_systemd_journal", Title: "ephemeral archive systemd journal", Metadata: strings.Join(journalPaths, ", "), Content: tail.Stdout})
					}
				}
			}
		}
	}
	if len(bundle.Sections) == 0 && len(bundle.Warnings) == 0 {
		bundle.Warnings = append(bundle.Warnings, fmt.Sprintf("no logs collected for %s", repoState.Repo.FullName))
	}
	return bundle
}

func nonEmptyLines(input string) []string {
	var out []string
	for _, line := range strings.Split(input, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
