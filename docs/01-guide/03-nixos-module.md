---
title: NixOS Module
description: Deploy one or more LinkPage instances declaratively with the flake's NixOS module.
---

# NixOS Module

The flake exports `nixosModules.default` for declarative LinkPage deployments.

## Import the module

```nix
inputs.linkpage.url = "github:rhnvrm/linkpage";

{ config, inputs, ... }:
{
  imports = [ inputs.linkpage.nixosModules.default ];
}
```

## Example instance

```nix
{ config, inputs, ... }:
{
  imports = [ inputs.linkpage.nixosModules.default ];

  services.linkpage.instances.personal = {
    enable = true;
    port = 8001;
    pageTitle = "Rohan Verma";
    pageIntro = "Builder, writer, and founder";
    pageLogoUrl = "/static/app/img/logos/logo-icon-only.png";

    social = {
      github = "https://github.com/rhnvrm";
      twitter = "https://x.com/rhnvrm";
    };

    declarative = true;
    links = [
      {
        url = "https://rohanverma.net";
        message = "Website";
        description = "Blog, projects, and notes";
        weight = 100;
      }
      {
        url = "https://github.com/rhnvrm";
        message = "GitHub";
        description = "Open-source projects";
        weight = 90;
      }
    ];

    auth = {
      username = "admin";
      passwordFile = config.age.secrets.linkpage-admin.path;
    };
  };
}
```

## Important details

- `declarative = true` makes the generated seed authoritative on every restart
- `links = [ ]` is valid and intentionally clears all links
- `auth.passwordFile` should point to a runtime secret path from agenix, sops-nix, or another secret manager
- The module keeps secrets out of the Nix store by loading the password through systemd credentials at runtime

## Service model

Each instance gets:

- its own systemd unit: `linkpage-<name>.service`
- a private state directory at `/var/lib/linkpage-<name>/`
- a generated config file and optional generated seed file

The module does not configure your public webserver. Add a reverse proxy entry in Caddy, Nginx, or another frontend for the domain you want to serve.
