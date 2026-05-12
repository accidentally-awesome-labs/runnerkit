package ops

import (
	"regexp"
	"strings"
)

const (
	HostHintKernelOOM          = "likely_kernel_oom"
	HostHintLinkerKill         = "likely_linker_sigkill"
	HostHintRunnerKill         = "likely_runner_hard_kill"
	HostHintRunnerMemorySignal = "likely_runner_memory_pressure"
)

// HostIncidentHint is a bounded heuristic from journal text (not a definitive diagnosis).
type HostIncidentHint struct {
	ID              string   `json:"id"`
	Severity        string   `json:"severity"`
	Summary         string   `json:"summary"`
	PatternsMatched []string `json:"patterns_matched,omitempty"`
	Snippets        []string `json:"snippets,omitempty"`
}

var (
	reKernelOOM  = regexp.MustCompile(`(?i)Out\s+of\s+memory|oom-kill|oom_reaper|Killed\s+process`)
	reSignalKill = regexp.MustCompile(`(?i)signal\s*9|SIGKILL|shutdown\s+signal`)
	reLinkerKill = regexp.MustCompile(`(?i)collect2:.*signal\s*9|ld\s+terminated\s+with\s+signal`)
	reRunnerOOM  = regexp.MustCompile(`(?i)Out\s+of\s+memory|oom-kill|cannot\s+allocate`)
)

// ShouldCollectHostIncidentJournals is true when SSH works and the runner is in a
// state where journal hints are likely actionable (or caller forced --deep).
func ShouldCollectHostIncidentJournals(obs ObservedRunner, deep bool) bool {
	if !obs.SSH.Reachable {
		return false
	}
	if deep {
		return true
	}
	if serviceFailed(obs.Service) {
		return true
	}
	if strings.EqualFold(obs.GitHub.Status, "offline") {
		return true
	}
	if !obs.GitHub.Found {
		return true
	}
	return false
}

// AnalyzeJournalForOOMHints scans runner and kernel journal text for common OOM /
// hard-kill signatures. maxSnippetLines is the max matching lines to retain per hint
// (0 disables snippets).
func AnalyzeJournalForOOMHints(runnerJournal, kernelJournal string, maxSnippetLines int) []HostIncidentHint {
	var hints []HostIncidentHint
	kernelPats := []struct {
		tag string
		re  *regexp.Regexp
	}{{"kernel_oom_or_kill", reKernelOOM}}
	if h := scanJournal(kernelJournal, HostHintKernelOOM, "warning",
		"Kernel ring buffer or journal shows OOM killer or memory pressure.",
		kernelPats, maxSnippetLines); h != nil {
		hints = append(hints, *h)
	}
	linkerPats := []struct {
		tag string
		re  *regexp.Regexp
	}{{"linker_signal_9", reLinkerKill}}
	if h := scanJournal(runnerJournal, HostHintLinkerKill, "warning",
		"Runner logs mention linker or collect2 killed with signal 9 (often OOM during native link).",
		linkerPats, maxSnippetLines); h != nil {
		hints = append(hints, *h)
	}
	runnerKillJournal := stripLinesMatching(runnerJournal, reLinkerKill)
	killPats := []struct {
		tag string
		re  *regexp.Regexp
	}{{"sigkill_or_shutdown", reSignalKill}}
	if h := scanJournal(runnerKillJournal, HostHintRunnerKill, "warning",
		"Runner logs mention SIGKILL, signal 9, or GitHub shutdown signal.",
		killPats, maxSnippetLines); h != nil {
		hints = append(hints, *h)
	}
	memPats := []struct {
		tag string
		re  *regexp.Regexp
	}{{"runner_memory_pressure", reRunnerOOM}}
	if h := scanJournal(runnerJournal, HostHintRunnerMemorySignal, "warning",
		"Runner logs mention memory pressure or OOM-related messages.",
		memPats, maxSnippetLines); h != nil {
		hints = append(hints, *h)
	}
	return hints
}

func stripLinesMatching(text string, exclude *regexp.Regexp) string {
	if exclude == nil {
		return text
	}
	var b strings.Builder
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if exclude.MatchString(line) {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func scanJournal(text, id, severity, summary string, patterns []struct {
	tag string
	re  *regexp.Regexp
}, maxSnippetLines int) *HostIncidentHint {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	var matched []string
	var snippets []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		for _, p := range patterns {
			if p.re.MatchString(line) {
				matched = append(matched, p.tag)
				if maxSnippetLines > 0 && len(snippets) < maxSnippetLines {
					s := line
					if len(s) > 240 {
						s = s[:237] + "..."
					}
					snippets = append(snippets, s)
				}
				break
			}
		}
	}
	if len(matched) == 0 {
		return nil
	}
	return &HostIncidentHint{
		ID:              id,
		Severity:        severity,
		Summary:         summary,
		PatternsMatched: matched,
		Snippets:        snippets,
	}
}
