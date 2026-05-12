package nextaction

import "testing"

func TestApplySchemaAndStage(t *testing.T) {
	p := ApplySchemaAndStage(map[string]any{"ok": true, "command": "doctor"}, "running")
	if p["schema_version"] != 1 || p["stage"] != "running" {
		t.Fatalf("%#v", p)
	}
}

func TestMergePayload_SetsSchemaVersion(t *testing.T) {
	p := MergePayload(map[string]any{"ok": true, "command": "x"}, "s", nil)
	if p["schema_version"] != 1 {
		t.Fatalf("schema_version = %v", p["schema_version"])
	}
	if p["stage"] != "s" {
		t.Fatalf("stage = %v", p["stage"])
	}
}

func TestInstallHostActions(t *testing.T) {
	a := InstallHostActions(`curl -fsSL "x" | sudo bash`)
	if len(a) != 1 || a[0].Kind != "run_on_host" {
		t.Fatalf("%+v", a)
	}
}
