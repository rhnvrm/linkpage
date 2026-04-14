---
title: CLI and Config
description: Runtime flags and core config fields.
---

# CLI and Config

## CLI flags

### `--init`

Creates starter files in the current directory.

```bash
./linkpage --init
```

### `--config`

Points LinkPage at a config file.

```bash
./linkpage --config /path/to/config.toml
```

### `--seed`

Applies a declarative seed file on startup.

```bash
./linkpage --config config.toml --seed seed.toml
```

## Core config fields

```toml
http_address = "0.0.0.0:8000"
read_timeout = "3s"
write_timeout = "3s"
dbfile = "app.db"

page_logo_url = "/static/app/img/logos/logo-icon-only.png"
page_title = "LinkPage"
page_intro = "Your friendly neighbourhood self-hostable FOSS link page."

static_files = "static/custom"

[auth]
username = "username"
password = "password"
# password_file = "/run/secrets/linkpage-admin-password"

[social]
github = "https://github.com/rhnvrm/linkpage"
twitter = "https://x.com/rhnvrm"
```

## Auth fields

- Use `auth.password` for local setups
- Use `auth.password_file` for systemd and NixOS deployments
- If `auth.password_file` is set, LinkPage reads the password from that file at startup

## Static overrides

Set `static_files` to a directory that should override embedded assets. Files missing there fall back to the built-in static files shipped in the binary.
