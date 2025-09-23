{
  inputs = {
    flake-parts.url = "github:hercules-ci/flake-parts";
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };
  outputs =
    inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [ ];
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
        "x86_64-darwin"
      ];
      perSystem =
        {
          config,
          self',
          inputs',
          pkgs,
          system,
          ...
        }:
        {
          formatter = pkgs.nixfmt-tree;
          devShells.default =
            with pkgs;
            mkShell {
              packages = [
                (writers.writeBashBin "develop-start-services" ''
                  ${pkgs.etcd}/bin/etcd \
                    --log-level warn \
                    --name .develop
                '')
                (writers.writeBashBin "develop-watch-clavelapi" ''
                  export PORT=8080
                  ${pkgs.watchexec}/bin/watchexec -e go -r "go run ./cmd/clavelapi"
                '')
                (writers.writeBashBin "develop-debug-clavelapi" ''
                  export PORT=8080
                  export CGO_CFLAGS="-O1"
                  ${pkgs.delve}/bin/dlv debug ./cmd/clavelapi
                '')
                (writers.writeBashBin "develop-watch-clavelcontroller" ''
                  ${pkgs.watchexec}/bin/watchexec -e go -r "go run ./cmd/clavelcontroller"
                '')
                buf
                delve
                etcd
                go
                gopls
                protoc-gen-connect-go
                protoc-gen-go
                watchexec
              ];
            };
        };
      flake = {
        flakeModule = ./flake-module.nix;
        lib = import ./lib.nix { nixpkgs-lib = inputs.nixpkgs.lib; };
      };
    };
}
