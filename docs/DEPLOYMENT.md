# Deployment

`disco serve` is a single Go binary with an embedded web client. This guide
covers running it as an unprivileged systemd service and mounting it behind
nginx, either at the root of a host or below a URL prefix such as `/disco`.

## Serve below a URL prefix

Use the same prefix in three places:

1. the nginx `location` and `X-Forwarded-Prefix` header,
2. optionally `--base-path` on the command line,
3. nowhere in the client: the server injects the prefix into `index.html` and
   the web client prefixes API, thumbnail, subtitle, HLS, and document URLs
   automatically.

```bash
disco serve /var/lib/disco/library.db --port 5555 --no-browser --base-path /disco
```

The prefix is also accepted from `X-Forwarded-Prefix`, so `--base-path` is
optional when nginx sets the header. The header is trusted when the immediate
peer is loopback; set `--trust-forwarded-prefix` when a non-loopback proxy must
supply it. Both proxy styles work:

- `proxy_pass http://127.0.0.1:5555;` keeps `/disco/...` in the request path;
  the server strips it.
- `proxy_pass http://127.0.0.1:5555/;` strips the prefix before forwarding;
  the header still tells the server which URLs to generate.

Invalid or malformed prefixes (control characters, `..` segments) are ignored,
and a request for `/` is redirected to the configured prefix.

An example nginx configuration is in
[`contrib/nginx/disco.conf`](../contrib/nginx/disco.conf). Keep
`proxy_buffering off`: the server streams server-sent events and HLS, and
buffering breaks both.

## systemd

```bash
sudo useradd --system --home /var/lib/disco --shell /usr/sbin/nologin disco
sudo install -d -o disco -g disco /var/lib/disco
sudo install -m 0755 disco /usr/local/bin/disco
sudo install -m 0644 contrib/systemd/disco.service /etc/systemd/system/disco.service
```

Edit `ExecStart` to list the databases you want to serve, then:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now disco
```

The unit in [`contrib/systemd/disco.service`](../contrib/systemd/disco.service):

- runs as the unprivileged `disco` user;
- caps the service cgroup with `MemoryHigh=70%` / `MemoryMax=80%`, so a
  memory-hungry transcode is OOM-killed instead of taking down the device;
- stops retrying after three failed starts in ten minutes;
- makes the filesystem read-only except the database directory, while media
  libraries stay readable.

`ReadWritePaths` must contain every directory that holds a database, not just
the database file: SQLite keeps `-wal` and `-shm` files beside it. Media
directories do not need write access.

If you use hardware transcoding, add the device nodes your GPU needs; do not
enable `PrivateDevices` for that setup.

## Tuning

All flags apply to the `serve` subcommand.

| Flag | Default | Purpose |
|---|---|---|
| `--host` | `127.0.0.1` | Listen address. A non-loopback address requires `DISCO_API_TOKEN`. |
| `--port` | `5555` | Listen port. |
| `--search-timeout` | `15s` | Upper bound for one `/api/query` request, including waiting for a search slot. |
| `--max-concurrent-searches` | `4` | Concurrent `/api/query` searches. Extra requests wait, then receive `503` when the timeout expires. Use `0` to disable the limit. |
| `--max-limit` | `1000` | Largest `limit` a client may request per database. Negative limits and offsets are ignored. |
| `--db-cache-mb` | `256` | SQLite page cache per connection. Lower it to roughly `32`-`64` on a 1 GB host. |
| `--read-only` | off | Disable progress updates, playlist changes, and deletions. |
| `--no-browser` | off | Required for headless deployments. |
| `--kiwix-host` | `127.0.0.1` | Host of the externally managed `kiwix-serve`. |
| `--kiwix-port` | `8181` | Port of the externally managed `kiwix-serve`. `0` disables ZIM support. |
| `--trust-forwarded-prefix` | off | Trust `X-Forwarded-Prefix` even when the peer is not loopback. |

Memory is dominated by the databases, the unbounded thumbnail cache, and any
`ffmpeg` transcodes. Measure the service rather than guessing:

```bash
systemctl status disco          # MemoryCurrent
systemd-cgtop                   # live per-unit memory
journalctl -u disco -f
```

On a small host, lower `--db-cache-mb`, keep `--max-concurrent-searches` at 1
or 2, and avoid serving multiple large databases at once.

## Multiple databases

Pass every database after `serve`. Queries fan out across all of them unless a
request filters with `--db` query parameters.

```ini
ExecStart=/usr/local/bin/disco serve \
    /var/lib/disco/library.db \
    /var/lib/disco/audio.db \
    /var/lib/disco/books.db \
    --port 5555 --no-browser --base-path /disco
```

## HTTPS and access

`disco` does not terminate TLS. Put it behind nginx or Caddy on the same host,
and keep the listen address at the default `127.0.0.1` (or another private
interface). The server refuses to bind to a non-loopback address unless
`DISCO_API_TOKEN` is set; a reverse proxy is the recommended remote deployment.

Token authentication is optional. A loopback-only server does not require a
token by default, so a local browser opens without a login prompt. Set
`DISCO_API_TOKEN` to require authentication; this is mandatory for non-loopback
binds.

The API token is not embedded in static assets. The client exchanges a token
for an `HttpOnly`, `SameSite=Strict` session cookie with `POST /api/session`.
Set `DISCO_API_TOKEN` in the service environment, then open the UI once with
`?token=<token>` or enter the token at the prompt. Scripts can keep using
`X-Disco-Token` or `Authorization: Bearer <token>`.

## External ZIM server

discoteca does not start, stop, wait for, or clean up `kiwix-serve`. Run it as a
separate service and tell discoteca which host and port to proxy to:

```bash
# Terminal 1: one or more ZIM files
kiwix-serve --port=8181 \
  --urlRootLocation=/api/zim/proxy/ \
  /var/lib/disco/library.zim

# Terminal 2
disco serve --kiwix-host 127.0.0.1 --kiwix-port 8181 /var/lib/disco/library.db
```

The external server must use the same `--urlRootLocation` as the proxy route.
If discoteca is mounted below a prefix, include it, for example
`--urlRootLocation=/disco/api/zim/proxy/` with `--base-path /disco`. The ZIM file
must also appear in a discoteca database; `/api/zim/view` rejects files that are
not listed. `--kiwix-port 0` disables ZIM support and returns `503` on ZIM
routes.

## Upgrades and reversibility

The binary is self-contained; an upgrade is replacing the file and restarting
the unit:

```bash
sudo install -m 0755 disco /usr/local/bin/disco
sudo systemctl restart disco
```

Downgrading the binary is safe for the web client because assets are embedded
in the binary. Database migrations are not automatically reversible, so back up
the database files (including the `-wal` file) before a major upgrade.
