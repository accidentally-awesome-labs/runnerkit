FROM ubuntu:24.04

# Minimal sshd host that simulates a fresh BYO machine.
# - One unprivileged user 'maintainer' with passworded sudo (mimics real hosts).
# - sshd listening so we can validate the SSH-as-non-root lifecycle after install.

RUN apt-get update -qq && \
    apt-get install -y --no-install-recommends \
      openssh-server sudo curl ca-certificates \
      iproute2 procps \
    && rm -rf /var/lib/apt/lists/*

# Create maintainer user with a known sudo password.
RUN useradd -m -s /bin/bash maintainer && \
    echo 'maintainer:dev-sudo-pwd' | chpasswd && \
    usermod -aG sudo maintainer

# sshd config: allow password auth + pubkey, listen on 22.
RUN mkdir -p /var/run/sshd && \
    sed -i 's|#PasswordAuthentication yes|PasswordAuthentication yes|' /etc/ssh/sshd_config && \
    sed -i 's|#PubkeyAuthentication yes|PubkeyAuthentication yes|' /etc/ssh/sshd_config && \
    sed -i 's|#PermitRootLogin .*|PermitRootLogin no|' /etc/ssh/sshd_config

# Inject the dev SSH pubkey (provided at build time as a buildarg).
ARG SSH_PUBKEY
RUN mkdir -p /home/maintainer/.ssh && \
    echo "${SSH_PUBKEY}" > /home/maintainer/.ssh/authorized_keys && \
    chmod 700 /home/maintainer/.ssh && \
    chmod 600 /home/maintainer/.ssh/authorized_keys && \
    chown -R maintainer:maintainer /home/maintainer/.ssh

EXPOSE 22

CMD ["/usr/sbin/sshd", "-D", "-e"]
