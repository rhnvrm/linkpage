---
title: Declarative Seed Mode
description: Make the database a cache of a TOML-defined link set.
---

# Declarative Seed Mode

Pass `--seed <file.toml>` to replace LinkPage's stored links from a TOML file on every startup.

```bash
./linkpage --config config.toml --seed seed.toml
```

Example:

```toml
[[links]]
url = "https://rohanverma.net"
message = "Personal Website"
description = "Blog, projects, and digital garden"
image_url = ""
weight = 100

[[links]]
url = "https://github.com/rhnvrm"
message = "GitHub"
description = "Open source projects"
image_url = ""
weight = 90
```

## Behavior

- The seed is authoritative at startup
- Existing links are replaced transactionally
- An empty seed is valid and clears all links
- Manual admin edits still work, but they will be overwritten on the next restart

## Migration from admin-managed mode

Seed mode is destructive by design, so switch carefully:

1. Back up `app.db`
2. Export or rewrite the current links into `seed.toml`
3. Start LinkPage once with `--seed`
4. Treat the database as runtime state, not the source of truth

## Operational guidance

- Keep the seed file in version control if you want reproducible deployments
- Use admin mode for experimentation, then copy the final links back into `seed.toml`
- Use [[NixOS Module]] if you want seed generation and service wiring handled declaratively
