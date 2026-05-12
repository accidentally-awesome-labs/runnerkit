package remote

import "testing"

func TestParseMeminfoAwkOutput(t *testing.T) {
	m, s, ok := ParseMeminfoAwkOutput("8123456 1048576")
	if !ok || m != 8123456 || s != 1048576 {
		t.Fatalf("ParseMeminfoAwkOutput = %d,%d ok=%v", m, s, ok)
	}
	_, _, ok = ParseMeminfoAwkOutput("garbage")
	if ok {
		t.Fatal("expected parse failure")
	}
}
