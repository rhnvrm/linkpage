{
  description = "LinkPage — self-hostable link-in-bio page";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "linkpage";
          version = "0.1.0-dev";
          src = ./.;
          vendorHash = "sha256-CjIhCo4/stwT+5EBQwrmaUC87ygnumyOIL7iJwgH0Ug=";
          env.CGO_ENABLED = 0;

          meta = {
            description = "Self-hostable link-in-bio page";
            homepage = "https://github.com/rhnvrm/linkpage";
            license = pkgs.lib.licenses.mit;
          };
        };
      }
    ) // {
      nixosModules.default = import ./nixos-module.nix self;
    };
}
