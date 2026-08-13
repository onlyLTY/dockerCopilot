# Docker Copilot daemon helper

This helper is for a Linux host running the local Docker Engine. It is deliberately separate from the Copilot container because changing `/etc/docker/daemon.json` and restarting Docker affects every container on the host.

Build and install:

```bash
cd helper
go build -o /usr/local/libexec/dockercopilot-helper .
cd ..
install -m 0644 helper/dockercopilot-helper.service /etc/systemd/system/
groupadd --system dockercopilot 2>/dev/null || true
systemctl daemon-reload
systemctl enable --now dockercopilot-helper.service
```

Enable the Copilot socket mount explicitly:

```bash
docker compose \
  -f docker/docker-compose.yml \
  -f docker/docker-compose.daemon-helper.yml \
  up -d
```

The helper accepts only fixed operations over `/run/dockercopilot-helper.sock`:

- Read daemon proxy status.
- Update only `proxies.http-proxy`, `proxies.https-proxy`, and `proxies.no-proxy`.
- Validate the daemon JSON and restart Docker with a fixed `systemctl restart docker` command.
- Report the asynchronous restart result.

The helper never accepts a filesystem path, shell command, or arbitrary daemon setting from the API. It backs up the previous configuration to `/var/lib/dockercopilot/daemon.json.bak` before an update. The page requires a dangerous confirmation for both writing the configuration and restarting Docker. Restarting Docker can interrupt Docker operations for every container on the host.

This helper is not supported for Docker Desktop, remote Docker daemons, rootless Docker, or Windows hosts. The default Compose file does not mount the helper socket, so the high-risk feature is disabled unless this override is explicitly used.
