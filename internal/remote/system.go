package remote

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// SystemExecutor uses the local ssh tooling so normal developer SSH config and agents work.
type SystemExecutor struct{}

func NewSystemExecutor() SystemExecutor { return SystemExecutor{} }

func (SystemExecutor) Probe(ctx context.Context, target Target) (ProbeResult, error) {
	hostKey := scanHostKey(ctx, target)
	probe := ProbeResult{HostKey: hostKey, Commands: map[string]bool{}}
	if kernel, err := sshOutput(ctx, target, "uname -s"); err == nil {
		probe.Kernel = strings.ToLower(strings.TrimSpace(kernel))
	} else {
		return probe, err
	}
	if arch, err := sshOutput(ctx, target, "uname -m"); err == nil {
		probe.Arch = strings.TrimSpace(arch)
	}
	if osRelease, err := sshOutput(ctx, target, "cat /etc/os-release 2>/dev/null || true"); err == nil {
		probe.OSRelease = parseOSRelease(osRelease)
	}
	if _, err := sshOutput(ctx, target, "test -d /run/systemd/system && command -v systemctl >/dev/null"); err == nil {
		probe.Systemd = true
	}
	for _, name := range []string{"sudo", "curl", "tar", "gzip", "sha256sum", "id", "useradd", "install", "timedatectl"} {
		if _, err := sshOutput(ctx, target, "command -v "+shellQuote(name)+" >/dev/null 2>&1"); err == nil {
			probe.Commands[name] = true
		}
	}
	if _, err := sshOutput(ctx, target, "sudo -n true >/dev/null 2>&1"); err == nil {
		probe.Commands["sudo"] = true
	}
	if disk, err := sshOutput(ctx, target, "df -PB1 /opt /var/lib 2>/dev/null | awk 'NR>1{if(min==0 || $4<min) min=$4} END{print min+0}'"); err == nil {
		probe.DiskAvailableBytes, _ = strconv.ParseInt(strings.TrimSpace(disk), 10, 64)
	}
	probe.MemAvailableBytes, probe.SwapFreeBytes = parseProcMeminfoViaAwk(ctx, target)
	if _, err := sshOutput(ctx, target, "timedatectl show -p NTPSynchronized --value 2>/dev/null | grep -qi true"); err == nil {
		probe.TimeSynchronized = true
	}
	return probe, nil
}

func (SystemExecutor) ProbeHostKey(ctx context.Context, target Target) (HostKey, error) {
	hostKey := scanHostKey(ctx, target)
	if NormalizeHostKey(hostKey).Fingerprint == "" {
		return hostKey, fmt.Errorf("SSH host key fingerprint was not observed")
	}
	return hostKey, nil
}

func (SystemExecutor) Run(ctx context.Context, target Target, command Command) (Result, error) {
	script := command.Script
	if len(command.Env) > 0 {
		var envLines []string
		for key, value := range command.Env {
			envLines = append(envLines, "export "+key+"="+shellQuote(value))
		}
		script = strings.Join(envLines, "\n") + "\n" + script
	}
	args := sshArgs(target, "bash -s")
	cmdCtx := ctx
	cancel := func() {}
	if command.Timeout > 0 {
		cmdCtx, cancel = context.WithTimeout(ctx, command.Timeout)
	}
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, "ssh", args...)
	cmd.Stdin = strings.NewReader(script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: 0}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
	}
	return result, err
}

func scanHostKey(ctx context.Context, target Target) HostKey {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ssh-keyscan", "-p", strconv.Itoa(target.Port), "-T", "5", target.Host)
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return HostKey{}
	}
	// Bug 24 (Plan 06-11, 2026-05-06): use selectHostKeyLine, which picks
	// a deterministic line regardless of the order ssh-keyscan emits the
	// server's keys. Without this, `up` and `status` could observe the
	// same host but pick different keys (ed25519 vs rsa) and produce
	// different fingerprints — falsely flagging `SSH ERROR host key
	// mismatch` despite the host being unchanged.
	line := selectHostKeyLine(string(out))
	fields := strings.Fields(line)
	algorithm := ""
	if len(fields) >= 2 {
		algorithm = fields[1]
	}
	return HostKey{Algorithm: algorithm, Fingerprint: FingerprintSHA256([]byte(line)), PublicKey: []byte(line)}
}

// SelectHostKeyLineForTest exposes selectHostKeyLine to tests in other
// packages so they can verify the host_key_match property end-to-end
// without duplicating the selection logic. It is a thin alias and not
// intended for production callers.
func SelectHostKeyLineForTest(output string) string { return selectHostKeyLine(output) }

// selectHostKeyLine picks one canonical line from ssh-keyscan output so
// that two separate scans of the same host always pick the same line —
// which keeps FingerprintSHA256 byte-stable across calls.
//
// Selection rules:
//  1. Skip blank lines and comments (#-prefixed).
//  2. Among remaining lines, prefer algorithms in this order:
//     ssh-ed25519, ecdsa-sha2-nistp521, ecdsa-sha2-nistp384,
//     ecdsa-sha2-nistp256, ssh-rsa, others.
//  3. Within the same algorithm precedence, fall back to lexicographic
//     ordering of the entire line so duplicate algorithms still resolve
//     deterministically.
//
// Returns empty string when output has no eligible line.
func selectHostKeyLine(output string) string {
	preference := map[string]int{
		"ssh-ed25519":         0,
		"ecdsa-sha2-nistp521": 1,
		"ecdsa-sha2-nistp384": 2,
		"ecdsa-sha2-nistp256": 3,
		"ssh-rsa":             4,
		"rsa-sha2-512":        5,
		"rsa-sha2-256":        6,
		"ssh-dss":             7,
	}
	bestRank := -1
	bestLine := ""
	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		rank, known := preference[fields[1]]
		if !known {
			rank = len(preference) // unknown algorithms rank after all known ones
		}
		if bestLine == "" {
			bestRank = rank
			bestLine = line
			continue
		}
		if rank < bestRank || (rank == bestRank && line < bestLine) {
			bestRank = rank
			bestLine = line
		}
	}
	return bestLine
}

// parseProcMeminfoViaAwk reads MemAvailable and SwapFree (kB) from /proc/meminfo
// and returns bytes. Returns (-1, -1) when values cannot be read.
func parseProcMeminfoViaAwk(ctx context.Context, target Target) (memBytes, swapBytes int64) {
	memBytes, swapBytes = -1, -1
	out, err := sshOutput(ctx, target, `awk '/^MemAvailable:/{m=$2} /^SwapFree:/{s=$2} END{print m+0,s+0}' /proc/meminfo 2>/dev/null || echo -1 -1`)
	if err != nil {
		return -1, -1
	}
	mkb, skb, ok := ParseMeminfoAwkOutput(out)
	if !ok {
		return -1, -1
	}
	if mkb >= 0 {
		memBytes = mkb * 1024
	}
	if skb >= 0 {
		swapBytes = skb * 1024
	}
	return memBytes, swapBytes
}

func sshOutput(ctx context.Context, target Target, script string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	args := sshArgs(target, script)
	cmd := exec.CommandContext(ctx, "ssh", args...)
	out, err := cmd.Output()
	return string(out), err
}

func sshArgs(target Target, remoteCommand string) []string {
	// Plan 06-16 (Bug 34): live Hetzner smoke hit persistent
	// `Host key verification failed` due stale entries in the operator's
	// global known_hosts when cloud IPs were recycled. RunnerKit already
	// probes and persists SSH host fingerprints in state, so system ssh
	// calls should not depend on ambient known_hosts trust. Disable
	// known_hosts file writes/reads here and rely on RunnerKit's explicit
	// host-key checks.
	args := []string{
		"-p", strconv.Itoa(target.Port),
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
	}
	if strings.TrimSpace(target.KeyPath) != "" {
		args = append(args, "-i", target.KeyPath)
	}
	args = append(args, target.User+"@"+target.Host, remoteCommand)
	return args
}

func parseOSRelease(input string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(input, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		out[parts[0]] = strings.Trim(strings.TrimSpace(parts[1]), `"`)
	}
	return out
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func (SystemExecutor) String() string { return fmt.Sprintf("system-ssh") }
