{
  description = "Multi-agent orchestration system for Claude Code with persistent work tracking";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    beads.url = "github:gastownhall/beads";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      beads,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };
        beadsPkg = beads.packages.${system}.default;
      in
      {
        packages = {
          lt = pkgs.buildGoModule {
            pname = "lt";
            version = "1.0.0";
            src = ./.;
            vendorHash = "sha256-mJzpsl4XnIm3ZSg7fFn0MOdQQW1bdOkAJ+TikiLMXJM=";

            ldflags = [
              "-X github.com/camp-leatherneck/camp-leatherneck/internal/cmd.Build=nix"
              "-X github.com/camp-leatherneck/camp-leatherneck/internal/cmd.BuiltProperly=1"
            ];

            subPackages = [ "cmd/lt" ];

            meta = with pkgs.lib; {
              description = "Multi-agent orchestration system for Claude Code with persistent work tracking";
              homepage = "https://github.com/camp-leatherneck/camp-leatherneck";
              license = licenses.mit;
              mainProgram = "lt";
            };
          };
          default = self.packages.${system}.lt;
        };

        apps = {
          lt = flake-utils.lib.mkApp {
            drv = self.packages.${system}.lt;
          };
          default = self.apps.${system}.lt;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = [
            beadsPkg
            pkgs.go_1_25
            pkgs.gopls
            pkgs.gotools
            pkgs.go-tools
          ];
        };
      }
    );
}
