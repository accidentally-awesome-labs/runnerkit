package remote

import "testing"

func TestParseTargetAcceptedForms(t *testing.T) {
	cases := []struct {
		raw  string
		user string
		host string
		port int
	}{
		{raw: "alice@example.com", user: "alice", host: "example.com", port: 22},
		{raw: "alice@example.com:2222", user: "alice", host: "example.com", port: 2222},
		{raw: "ssh://alice@example.com:2222", user: "alice", host: "example.com", port: 2222},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			target, err := ParseTarget(tc.raw, 22)
			if err != nil {
				t.Fatalf("ParseTarget returned error: %v", err)
			}
			if target.User != tc.user || target.Host != tc.host || target.Port != tc.port {
				t.Fatalf("target = %#v, want user=%s host=%s port=%d", target, tc.user, tc.host, tc.port)
			}
			if target.Display() != tc.user+"@"+tc.host+":"+itoa(tc.port) {
				t.Fatalf("Display() = %q", target.Display())
			}
		})
	}
}

func TestParseTargetRejectsInvalidForms(t *testing.T) {
	for _, raw := range []string{"example.com", "@example.com", "alice@", "alice@example.com:not-a-port", "alice@example.com:70000"} {
		t.Run(raw, func(t *testing.T) {
			if _, err := ParseTarget(raw, 22); err == nil {
				t.Fatalf("ParseTarget(%q) succeeded, want error", raw)
			}
		})
	}
}

func itoa(value int) string {
	if value == 22 {
		return "22"
	}
	if value == 2222 {
		return "2222"
	}
	return ""
}

func TestCanonicalHostKeyNormalizesPort22(t *testing.T) {
	a, err := CanonicalHostKey("alice@example.com", 22)
	if err != nil {
		t.Fatal(err)
	}
	b, err := CanonicalHostKey("alice@example.com:22", 22)
	if err != nil {
		t.Fatal(err)
	}
	if a != b || a != "alice@example.com:22" {
		t.Fatalf("CanonicalHostKey mismatch: a=%q b=%q", a, b)
	}
}
