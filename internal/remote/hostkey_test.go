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
