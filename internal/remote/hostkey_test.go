package remote

import (
	"strings"
	"testing"
)

func TestFingerprintSHA256Shape(t *testing.T) {
	fingerprint := FingerprintSHA256([]byte("fake ssh public key bytes"))
	if !strings.HasPrefix(fingerprint, "SHA256:") {
		t.Fatalf("fingerprint = %q, want SHA256: prefix", fingerprint)
	}
	if strings.Contains(fingerprint, "=") {
		t.Fatalf("fingerprint includes base64 padding: %q", fingerprint)
	}
}

// Bug 24 (Plan 06-11, 2026-05-06): when an SSH server publishes multiple
// host keys (the default Ubuntu 24.04 sshd serves ed25519 + ecdsa + rsa),
// `ssh-keyscan` returns the keys in whatever order the server happens
// to emit them. That order is NOT guaranteed to be stable across calls
// — we observed a live mismatch where `runnerkit up` saved the
// ed25519 fingerprint (because ssh-keyscan emitted ed25519 first) and a
// later `runnerkit status` reconnected, got the rsa key first, and
// reported `SSH ERROR host key mismatch` despite the host being
// unchanged.
//
// The fix is to make selectHostKeyLine deterministic by preferring
// algorithms in a stable order (ed25519 > ecdsa > rsa > others) and
// falling back to lexicographic order within the same precedence. Both
// `up` and `status` go through the same selector, so identical
// ssh-keyscan output (regardless of internal line order) collapses to
// the same chosen line, and FingerprintSHA256 of that line is byte-equal
// across calls — host_key_match property restored.
func TestSelectHostKeyLineIsDeterministicAcrossKeyOrders(t *testing.T) {
	rsaLine := "[mckee-small-desktop]:22 ssh-rsa AAAARSA-fake-key-bytes"
	ecdsaLine := "[mckee-small-desktop]:22 ecdsa-sha2-nistp256 AAAAECDSA-fake-key-bytes"
	ed25519Line := "[mckee-small-desktop]:22 ssh-ed25519 AAAAED25519-fake-key-bytes"

	// Order A: as `up` saw it.
	orderA := strings.Join([]string{rsaLine, ecdsaLine, ed25519Line}, "\n") + "\n"
	// Order B: as `status` saw it on the next call (server emitted in different order).
	orderB := strings.Join([]string{ed25519Line, rsaLine, ecdsaLine}, "\n") + "\n"
	// Order C: rsa-first.
	orderC := strings.Join([]string{ecdsaLine, ed25519Line, rsaLine}, "\n") + "\n"

	pickA := selectHostKeyLine(orderA)
	pickB := selectHostKeyLine(orderB)
	pickC := selectHostKeyLine(orderC)
	if pickA != pickB || pickB != pickC {
		t.Fatalf("selectHostKeyLine must be deterministic across input orders;\n  A=%q\n  B=%q\n  C=%q", pickA, pickB, pickC)
	}
	// Preference order: ed25519 wins when present.
	if pickA != ed25519Line {
		t.Fatalf("selectHostKeyLine must prefer ed25519; got %q want %q", pickA, ed25519Line)
	}

	// host_key_match: FingerprintSHA256 of the chosen line is byte-equal
	// regardless of which order ssh-keyscan emitted on either call.
	if FingerprintSHA256([]byte(pickA)) != FingerprintSHA256([]byte(pickB)) {
		t.Fatal("FingerprintSHA256 differs between equivalent ssh-keyscan outputs in different orders — host_key_match property broken")
	}
}

// When ed25519 is absent, prefer ecdsa-sha2-nistp256 next.
func TestSelectHostKeyLinePrefersEcdsaWhenNoEd25519(t *testing.T) {
	rsaLine := "[host]:22 ssh-rsa AAAARSA"
	ecdsaLine := "[host]:22 ecdsa-sha2-nistp256 AAAAECDSA"
	got := selectHostKeyLine(strings.Join([]string{rsaLine, ecdsaLine}, "\n") + "\n")
	if got != ecdsaLine {
		t.Fatalf("selectHostKeyLine must prefer ecdsa over rsa when ed25519 is absent; got %q want %q", got, ecdsaLine)
	}
}

// Comment lines + blanks must be skipped, not chosen.
func TestSelectHostKeyLineSkipsCommentsAndBlanks(t *testing.T) {
	input := "# comment line\n\n[host]:22 ssh-ed25519 AAAAED\n# more comment\n"
	got := selectHostKeyLine(input)
	if got != "[host]:22 ssh-ed25519 AAAAED" {
		t.Fatalf("comments must be skipped; got %q", got)
	}
}

// Single-line input (one algorithm only) returns that line unchanged.
func TestSelectHostKeyLineSingleLineIsPassThrough(t *testing.T) {
	line := "[host]:22 ssh-rsa AAAARSA"
	got := selectHostKeyLine(line + "\n")
	if got != line {
		t.Fatalf("single-line passthrough; got %q want %q", got, line)
	}
}

func TestDecideHostKeyStates(t *testing.T) {
	key := HostKey{Algorithm: "ssh-ed25519", Fingerprint: "SHA256:observed"}
	if decision, err := DecideHostKey("", key); err != nil || decision != HostKeyUnknown {
		t.Fatalf("unknown decision = %q err=%v", decision, err)
	}
	if decision, err := DecideHostKey("SHA256:observed", key); err != nil || decision != HostKeyAccepted {
		t.Fatalf("accepted decision = %q err=%v", decision, err)
	}
	decision, err := DecideHostKey("SHA256:stored", key)
	if err == nil || decision != HostKeyMismatch || !strings.Contains(err.Error(), HostKeyMismatchCode) {
		t.Fatalf("mismatch decision=%q err=%v", decision, err)
	}
}
