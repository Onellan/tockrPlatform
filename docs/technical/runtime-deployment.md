# Platform runtime deployment

PF-B10-S01 defines the first executable Platform runtime hardening surface.
The process is one Go/Chi application backed by the existing single SQLite
connection and a persistent `platform.db` volume. Platform does not mount or
read CTRL or IMS databases.

## Configuration

The process validates `PLATFORM_MFA_KEY` as exactly 64 hexadecimal characters.
`PLATFORM_DB_PATH` and `PLATFORM_HTTP_ADDR` have safe local defaults; the
container supplies `/var/lib/tockrplatform/platform.db` and `:8080`. Boolean
configuration accepts only `0` or `1`, and configuration errors never echo
secret material. Assertion configuration remains required at startup.

## HTTP lifecycle and boundaries

The server applies bounded request bodies, header and connection timeouts,
restrictive security headers, safe external error messages and a bounded
graceful shutdown. `/healthz` is a dependency-free liveness response. `/readyz`
checks the Platform SQLite handle and returns `503` without exposing the
underlying error when the dependency is unavailable. Both responses are
uncached and contain no tenant or secret data.

## Container contract

`Dockerfile` is a multi-stage, cross-platform build for Linux AMD64 and ARM64.
The final image runs as UID/GID `65532`, has no shell dependency, and writes
only to the persistent `/var/lib/tockrplatform` volume. `compose.yaml` applies
read-only root storage, a bounded `tmpfs`, dropped capabilities and
`no-new-privileges`. Secrets are supplied through the deployment environment,
never baked into the image.

The container build profiles in `scripts/validate.py` are required evidence for
this Slice. An unavailable Docker daemon is an environment failure and remains
blocked; it is not treated as a successful build.
