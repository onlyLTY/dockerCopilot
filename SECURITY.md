# Security notes

## Deployment boundary

Docker Copilot controls the Docker daemon through its socket. Access to the
application therefore has host-equivalent administrative impact. Bind the
management port to a trusted network, use HTTPS, protect `secretKey`, and do
not expose the service directly to the public Internet.

Only configure `TRUSTED_PROXY_CIDRS` for reverse proxies you administer. The
application ignores forwarded client-address headers from every other peer.

## Dependency advisories

`govulncheck` currently reports `GO-2026-4887` and `GO-2026-4883` through the
Moby module. No fixed Moby release is available at the time of this review.
Both advisories affect Docker daemon AuthZ/plugin server behavior; Docker
Copilot imports and uses the Docker client API and does not run those server
components. The dependency should still be upgraded as soon as a compatible
fixed release becomes available.

## Backups and credentials

JSON backups are encrypted. Compose exports omit sensitive environment values
by default, but are plaintext when `COMPOSE_BACKUP_INCLUDE_SECRETS=true`.
Registry credentials are read from Docker auth configuration and are never
included in API responses.
