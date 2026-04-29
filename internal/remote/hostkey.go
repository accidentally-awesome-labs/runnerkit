package remote

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

type HostKey struct {
	Algorithm   string
	Fingerprint string
	PublicKey   []byte
}

type HostKeyDecision string

const (
	HostKeyAccepted HostKeyDecision = "accepted"
	HostKeyUnknown  HostKeyDecision = "unknown"
	HostKeyMismatch HostKeyDecision = "mismatch"

	HostKeyMismatchCode = "ssh_host_key_mismatch"
)

type HostKeyMismatchError struct {
	Expected string
	Observed string
}

func (e HostKeyMismatchError) Error() string {
	return fmt.Sprintf("%s: expected %s, observed %s", HostKeyMismatchCode, e.Expected, e.Observed)
}

func FingerprintSHA256(publicKey []byte) string {
	sum := sha256.Sum256(publicKey)
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(sum[:])
}

func NormalizeHostKey(key HostKey) HostKey {
	key.Algorithm = strings.TrimSpace(key.Algorithm)
	key.Fingerprint = strings.TrimSpace(key.Fingerprint)
	if key.Fingerprint == "" && len(key.PublicKey) > 0 {
		key.Fingerprint = FingerprintSHA256(key.PublicKey)
	}
	return key
}

func DecideHostKey(storedFingerprint string, observed HostKey) (HostKeyDecision, error) {
	observed = NormalizeHostKey(observed)
	storedFingerprint = strings.TrimSpace(storedFingerprint)
	if storedFingerprint == "" {
		return HostKeyUnknown, nil
	}
	if observed.Fingerprint == storedFingerprint {
		return HostKeyAccepted, nil
	}
	return HostKeyMismatch, HostKeyMismatchError{Expected: storedFingerprint, Observed: observed.Fingerprint}
}
