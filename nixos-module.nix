flake:
{ config, lib, pkgs, ... }:

let
  cfg = config.services.linkpage;

  linkOpts = lib.types.submodule {
    options = {
      url = lib.mkOption {
        type = lib.types.str;
        description = "Link URL";
      };
      message = lib.mkOption {
        type = lib.types.str;
        description = "Link display text";
      };
      description = lib.mkOption {
        type = lib.types.str;
        default = "";
        description = "Link description";
      };
      image_url = lib.mkOption {
        type = lib.types.str;
        default = "";
        description = "Link image URL";
      };
      weight = lib.mkOption {
        type = lib.types.int;
        default = 0;
        description = "Sort weight (higher = first)";
      };
    };
  };

  instanceOpts = { name, ... }: {
    options = {
      enable = lib.mkEnableOption "LinkPage instance ${name}";

      port = lib.mkOption {
        type = lib.types.port;
        default = 8000;
        description = "HTTP port to listen on";
      };

      pageTitle = lib.mkOption {
        type = lib.types.str;
        default = "LinkPage";
        description = "Page title";
      };

      pageIntro = lib.mkOption {
        type = lib.types.str;
        default = "";
        description = "Page intro text";
      };

      pageLogoUrl = lib.mkOption {
        type = lib.types.str;
        default = "/static/app/img/logos/logo-icon-only.png";
        description = "Page logo URL";
      };

      social = lib.mkOption {
        type = lib.types.attrsOf lib.types.str;
        default = {};
        description = "Social media links (key = platform, value = URL)";
      };

      links = lib.mkOption {
        type = lib.types.listOf linkOpts;
        default = [];
        description = "Declarative link entries";
      };

      auth = {
        username = lib.mkOption {
          type = lib.types.str;
          default = "admin";
          description = "Admin panel username";
        };
        password = lib.mkOption {
          type = lib.types.str;
          default = "changeme";
          description = "Admin panel password";
        };
      };

      package = lib.mkOption {
        type = lib.types.package;
        default = flake.packages.${pkgs.stdenv.hostPlatform.system}.default;
        description = "LinkPage package to use";
      };
    };
  };

  # Generate config.toml for an instance
  mkConfigToml = name: inst: pkgs.writeText "linkpage-${name}-config.toml" ''
    http_address = "127.0.0.1:${toString inst.port}"
    read_timeout = "3s"
    write_timeout = "3s"
    dbfile = "/var/lib/linkpage-${name}/app.db"

    page_logo_url = "${inst.pageLogoUrl}"
    page_title = "${inst.pageTitle}"
    page_intro = "${inst.pageIntro}"

    static_files = ""

    [auth]
    username = "${inst.auth.username}"
    password = "${inst.auth.password}"

    ${lib.optionalString (inst.social != {}) ''
    [social]
    ${lib.concatStringsSep "\n" (lib.mapAttrsToList (k: v: ''${k} = "${v}"'') inst.social)}
    ''}
  '';

  # Generate seed.toml for an instance
  mkSeedToml = name: inst: pkgs.writeText "linkpage-${name}-seed.toml" (
    lib.concatStringsSep "\n" (map (link: ''
      [[links]]
      url = "${link.url}"
      message = "${link.message}"
      description = "${link.description}"
      image_url = "${link.image_url}"
      weight = ${toString link.weight}
    '') inst.links)
  );

  enabledInstances = lib.filterAttrs (_: inst: inst.enable) cfg.instances;

in
{
  options.services.linkpage = {
    instances = lib.mkOption {
      type = lib.types.attrsOf (lib.types.submodule instanceOpts);
      default = {};
      description = "LinkPage instances";
    };
  };

  config = lib.mkIf (enabledInstances != {}) {
    systemd.services = lib.mapAttrs' (name: inst:
      lib.nameValuePair "linkpage-${name}" {
        description = "LinkPage - ${name}";
        after = [ "network.target" ];
        wantedBy = [ "multi-user.target" ];

        serviceConfig = {
          Type = "simple";
          DynamicUser = true;
          StateDirectory = "linkpage-${name}";
          ExecStart = let
            configFile = mkConfigToml name inst;
            seedArgs = lib.optionalString (inst.links != [])
              " --seed ${mkSeedToml name inst}";
          in "${inst.package}/bin/linkpage --config ${configFile}${seedArgs}";
          Restart = "on-failure";
          RestartSec = 5;

          # Hardening
          ProtectSystem = "strict";
          ProtectHome = true;
          PrivateTmp = true;
          NoNewPrivileges = true;
        };
      }
    ) enabledInstances;
  };
}
