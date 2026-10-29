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
optional when nginx sets the header. Both proxy styles work:

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
| `--search-timeout` | `15s` | Upper bound for one `/api/query` request, including waiting for a search slot. |
| `--max-concurrent-searches` | `4` | Concurrent `/api/query` searches. Extra requests wait, then receive `503` when the timeout expires. Use `0` to disable the limit. |
| `--max-limit` | `1000` | Largest `limit` a client may request per database. Negative limits and offsets are ignored. |
| `--db-cache-mb` | `256` | SQLite page cache per connection. Lower it to roughly `32`-`64` on a 1 GB host. |
| `--read-only` | off | Disable progress updates, playlist changes, and deletions. |
| `--port` | `5555` | Listen port. |
| `--no-browser` | off | Required for headless deployments. |

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
and keep the listen port bound to loopback or a private interface. The server
sets an `HttpOnly`, `SameSite=Strict` token cookie scoped to the configured
base path; do not expose the port directly to the internet.

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
