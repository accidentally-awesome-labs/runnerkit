#!/usr/bin/env bash
# scripts/testkit/host-container.sh: a disposable password-sudo Ubuntu 24.04
# host in a privileged systemd container, for the release gate on a Linux
# x86_64 workstation with Docker. A fresh cloud VM works just as well and is
# the only option on macOS (docs/testkit/README.md).
#
#   scripts/testkit/host-container.sh up [--name NAME] [--port PORT] [--user USER] [--pubkey FILE]
#   scripts/testkit/host-container.sh down [--name NAME]
#
# The host has only what a minimal Ubuntu server has plus openssh-server,
# sudo and curl, and USER is in the sudo group with a password (printed
# once). Nothing grants passwordless sudo: install.sh must do that.
#
# The container is privileged because Docker inside it needs that, so a job
# on its runner is root on this workstation's kernel. Run only your own
# workflows from the throwaway private repository.
set -euo pipefail

name=rk-gate-host
port=2222
user=rkadmin
pubkey=""
action=""
while [ $# -gt 0 ]; do
	case "$1" in
	up | down)
		action="$1"
		shift
		;;
	--name)
		name="${2:?--name needs a value}"
		shift 2
		;;
	--port)
		port="${2:?--port needs a value}"
		shift 2
		;;
	--user)
		user="${2:?--user needs a value}"
		shift 2
		;;
	--pubkey)
		pubkey="${2:?--pubkey needs a value}"
		shift 2
		;;
	-h | --help)
		sed -n '2,17p' "$0"
		exit 0
		;;
	*)
		echo "unknown argument: $1" >&2
		exit 2
		;;
	esac
done
[ -n "$action" ] || {
	echo "usage: $0 up|down [--name NAME] [--port PORT] [--user USER] [--pubkey FILE]" >&2
	exit 2
}
command -v docker >/dev/null 2>&1 || {
	echo "FAIL: docker is required" >&2
	exit 1
}

if [ "$action" = down ]; then
	docker rm -f "$name" >/dev/null 2>&1 || true
	docker volume rm "$name-docker" >/dev/null 2>&1 || true
	echo "Removed container $name and volume $name-docker."
	exit 0
fi

if [ "$(uname -s)-$(uname -m)" != Linux-x86_64 ]; then
	echo "FAIL: this needs a Linux x86_64 Docker host (RunnerKit supports only x86_64 runner hosts); use a cloud VM instead" >&2
	exit 1
fi
if [ -z "$pubkey" ]; then
	for candidate in "$HOME/.ssh/id_ed25519.pub" "$HOME/.ssh/id_ecdsa.pub" "$HOME/.ssh/id_rsa.pub"; do
		if [ -f "$candidate" ]; then
			pubkey="$candidate"
			break
		fi
	done
fi
[ -n "$pubkey" ] && [ -f "$pubkey" ] || {
	echo "FAIL: no SSH public key found; pass --pubkey FILE" >&2
	exit 1
}
if docker inspect "$name" >/dev/null 2>&1; then
	echo "FAIL: container $name exists; run '$0 down --name $name' first for a fresh host" >&2
	exit 1
fi

image=runnerkit-testkit-host:24.04
docker build -q -t "$image" - >/dev/null <<'EOF'
FROM ubuntu:24.04
ENV container=docker DEBIAN_FRONTEND=noninteractive
RUN apt-get update \
 && apt-get install -y --no-install-recommends systemd systemd-sysv dbus openssh-server sudo curl ca-certificates tzdata \
 && rm -rf /var/lib/apt/lists/* \
 && rm -f /usr/sbin/policy-rc.d \
 && systemctl enable ssh \
 && systemctl mask getty.target console-getty.service systemd-logind.service
STOPSIGNAL SIGRTMIN+3
CMD ["/sbin/init"]
EOF

docker run -d --name "$name" --hostname "$name" --privileged --cgroupns=host \
	-v /sys/fs/cgroup:/sys/fs/cgroup:rw --tmpfs /run --tmpfs /run/lock \
	-v "$name-docker:/var/lib/docker" -p "127.0.0.1:$port:22" "$image" >/dev/null

i=0
until docker exec "$name" systemctl is-system-running 2>/dev/null | grep -Eq 'running|degraded'; do
	i=$((i + 1))
	if [ "$i" -ge 60 ]; then
		echo "FAIL: systemd did not start in $name" >&2
		exit 1
	fi
	sleep 1
done

password="$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 16 || true)"
docker exec "$name" useradd -m -s /bin/bash -G sudo "$user"
printf '%s:%s\n' "$user" "$password" | docker exec -i "$name" chpasswd
docker exec "$name" install -d -m 700 -o "$user" -g "$user" "/home/$user/.ssh"
docker exec -i "$name" install -m 600 -o "$user" -g "$user" /dev/stdin "/home/$user/.ssh/authorized_keys" <"$pubkey"
# Fresh SSH host keys for this container (the image's are shared).
docker exec "$name" sh -c 'rm -f /etc/ssh/ssh_host_* && ssh-keygen -A >/dev/null && systemctl restart ssh && rm -f /run/nologin'

cat <<EOF
Host $name is up: Ubuntu 24.04 x86_64 with systemd, sshd on 127.0.0.1:$port.
  SSH:            ssh -p $port $user@127.0.0.1
  sudo password:  $password   (shown once; sudo asks for it until install.sh runs)
  RunnerKit host: $user@127.0.0.1:$port
Remove it with: $0 down --name $name
EOF
