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
	line := firstHostKeyLine(string(out))
	fields := strings.Fields(line)
	algorithm := ""
	if len(fields) >= 2 {
		algorithm = fields[1]
	}
	return HostKey{Algorithm: algorithm, Fingerprint: FingerprintSHA256([]byte(line)), PublicKey: []byte(line)}
}

func firstHostKeyLine(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return line
	}
	return strings.TrimSpace(output)
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
	args := []string{"-p", strconv.Itoa(target.Port), "-o", "BatchMode=yes", "-o", "ConnectTimeout=10"}
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
