package ops

import "testing"

func TestAnalyzeJournalForOOMHints_KernelOOM(t *testing.T) {
	h := AnalyzeJournalForOOMHints("", "Jun 1 00:00:00 host kernel: Out of memory: Killed process 42 (rustc)\n", 2)
	if len(h) < 1 || h[0].ID != HostHintKernelOOM {
		t.Fatalf("hints=%#v", h)
	}
	if len(h[0].Snippets) < 1 {
		t.Fatalf("expected snippets, got %#v", h)
	}
}

func TestAnalyzeJournalForOOMHints_Linker(t *testing.T) {
	h := AnalyzeJournalForOOMHints("collect2: fatal error: ld terminated with signal 9 [Killed]\n", "", 0)
	if len(h) != 1 || h[0].ID != HostHintLinkerKill {
		t.Fatalf("hints=%#v", h)
	}
}

func TestAnalyzeJournalForOOMHints_Empty(t *testing.T) {
	if h := AnalyzeJournalForOOMHints("all quiet\n", "nothing\n", 3); len(h) != 0 {
		t.Fatalf("expected no hints, got %#v", h)
	}
}

func TestShouldCollectHostIncidentJournals(t *testing.T) {
	obs := ObservedRunner{SSH: SSHFact{Reachable: true}, GitHub: GitHubFact{Found: true, Status: "online"}, Service: ServiceFact{ActiveState: "active"}}
	if ShouldCollectHostIncidentJournals(obs, false) {
		t.Fatal("healthy runner should not collect unless deep")
	}
	if !ShouldCollectHostIncidentJournals(obs, true) {
		t.Fatal("deep should collect when SSH reachable")
	}
	obs.Service.ActiveState = "failed"
	if !ShouldCollectHostIncidentJournals(obs, false) {
		t.Fatal("failed service should collect")
	}
}
