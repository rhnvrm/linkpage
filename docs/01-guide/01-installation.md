---
title: Installation
description: Run LinkPage from a release artifact, Docker image, source checkout, or Nix.
---

# Installation

## Pre-built binary

1. Download the latest archive from the [releases page](https://github.com/rhnvrm/linkpage/releases)
2. Extract it
3. Initialize the app in an empty directory:

```bash
./linkpage --init
```

4. Start LinkPage:

```bash
./linkpage
```

Open <http://localhost:8000> and sign in at <http://localhost:8000/admin>.

Default credentials after `--init` come from the generated `config.toml`. Change them before exposing the app publicly.

## Docker

Initialize config and database:

```bash
docker run -v linkpage:/linkpage -p 8000:8000 rhnvrm/linkpage:latest ./linkpage --init
```

Run the app:

```bash
docker run -v linkpage:/linkpage -p 8000:8000 rhnvrm/linkpage:latest ./linkpage
```

## Run from source

Requirements:
- Go 1.21+

```bash
git clone https://github.com/rhnvrm/linkpage.git
cd linkpage
cp config.sample.toml config.toml
go run .
```

## Build with Nix

```bash
nix build github:rhnvrm/linkpage
./result/bin/linkpage --config config.toml
```

To start in declarative mode, add a seed file:

```bash
./result/bin/linkpage --config config.toml --seed seed.toml
```

## Next steps

- Continue with [[Declarative Seed Mode]] if you want links managed from a file
- Continue with [[NixOS Module]] if you want to run LinkPage as a declarative service
