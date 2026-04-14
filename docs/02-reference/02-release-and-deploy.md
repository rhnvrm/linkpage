---
title: Release and Deploy
description: How this repository publishes binaries, docs, and deployable artifacts.
---

# Release and Deploy

## Application releases

LinkPage uses GoReleaser from `.github/workflows/release.yml`.

Current release behavior:

- push a git tag such as `v0.5.0`
- GitHub Actions builds release archives
- checksums are attached to the GitHub release
- Docker images are published to `rhnvrm/linkpage`

## Documentation site

The docs in `docs/` are built with [moat](https://github.com/oddship/moat) and deployed with GitHub Pages using `.github/workflows/docs.yml`.

Typical setup:

1. In GitHub repository settings, open **Pages**
2. Set **Source** to **GitHub Actions**
3. Push changes to `docs/` on the default branch

The docs site is expected at:
- <https://rhnvrm.github.io/linkpage/>

## Nix and production deployment

For Nix-based deployments, this repo provides:

- a flake package for `nix build`
- a NixOS module for one or more LinkPage instances
- declarative seed support so the database can be treated as runtime state

A production deployment typically needs three layers:

1. LinkPage package or binary
2. LinkPage service configuration and secrets
3. Reverse proxy and DNS for the public hostname

Use [[NixOS Module]] for the service layer, then add your site-specific reverse proxy config separately.
