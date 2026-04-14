flake:
{ config, lib, pkgs, ... }:

let
  cfg = config.services.linkpage;
  tomlFormat = pkgs.formats.toml { };

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
        default = { };
        description = "Social media links (key = platform, value = URL)";
      };

      declarative = lib.mkOption {
        type = lib.types.bool;
        default = false;
        description = ''
          Manage links declaratively via a generated seed file. When enabled,
          LinkPage replaces database links on every startup, including an empty
          list to intentionally clear all links.
        '';
      };

      links = lib.mkOption {
        type = lib.types.listOf linkOpts;
        default = [ ];
        description = "Declarative link entries";
      };

      auth = {
        username = lib.mkOption {
          type = lib.types.str;
          default = "admin";
          description = "Admin panel username";
        };
        passwordFile = lib.mkOption {
          type = lib.types.nullOr lib.types.str;
          default = null;
          description = ''
            Path to a runtime secret file containing the admin password.
            Use agenix, sops-nix, or another secret manager. The module writes
            only this path to the Nix store, never the password itself.
          '';
        };
      };

      package = lib.mkOption {
        type = lib.types.package;
        default = flake.packages.${pkgs.stdenv.hostPlatform.system}.default;
        description = "LinkPage package to use";
      };
    };
  };

  credentialName = "linkpage-admin-password";
  passwordFilePlaceholder = "__LINKPAGE_PASSWORD_FILE__";

  mkConfigToml = name: inst:
    tomlFormat.generate "linkpage-${name}-config.toml" ({
      http_address = "127.0.0.1:${toString inst.port}";
      read_timeout = "3s";
      write_timeout = "3s";
      dbfile = "/var/lib/linkpage-${name}/app.db";
      page_logo_url = inst.pageLogoUrl;
      page_title = inst.pageTitle;
      page_intro = inst.pageIntro;
      static_files = "";
      auth = {
        username = inst.auth.username;
        password_file = passwordFilePlaceholder;
      };
    } // lib.optionalAttrs (inst.social != { }) {
      social = inst.social;
    });

  mkSeedToml = name: inst:
    tomlFormat.generate "linkpage-${name}-seed.toml" {
      links = inst.links;
    };

  shouldSeed = inst: inst.declarative || inst.links != [ ];

  enabledInstances = lib.filterAttrs (_: inst: inst.enable) cfg.instances;

in
{
  options.services.linkpage = {
    instances = lib.mkOption {
      type = lib.types.attrsOf (lib.types.submodule instanceOpts);
      default = { };
      description = "LinkPage instances";
    };
  };

  config = lib.mkIf (enabledInstances != { }) {
    assertions = lib.mapAttrsToList (name: inst: {
      assertion = inst.auth.passwordFile != null;
      message = "services.linkpage.instances.${name}.auth.passwordFile must be set to a runtime secret path";
    }) enabledInstances;

    systemd.services = lib.mapAttrs' (name: inst:
      lib.nameValuePair "linkpage-${name}" {
        description = "LinkPage - ${name}";
        after = [ "network.target" ];
        wantedBy = [ "multi-user.target" ];

        serviceConfig = let
          configFile = mkConfigToml name inst;
          seedArgs = lib.optionalString (shouldSeed inst)
            " --seed ${mkSeedToml name inst}";
          startScript = pkgs.writeShellScript "linkpage-${name}-start" ''
            runtime_config="$RUNTIME_DIRECTORY/config.toml"
            ${pkgs.gnused}/bin/sed \
              "s#${passwordFilePlaceholder}#$CREDENTIALS_DIRECTORY/${credentialName}#g" \
              ${configFile} > "$runtime_config"
            exec ${inst.package}/bin/linkpage --config "$runtime_config"${seedArgs}
          '';
        in {
          Type = "simple";
          DynamicUser = true;
          StateDirectory = "linkpage-${name}";
          RuntimeDirectory = "linkpage-${name}";
          RuntimeDirectoryMode = "0700";
          LoadCredential = "${credentialName}:${inst.auth.passwordFile}";
          ExecStart = startScript;
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
