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
                (writers.writeBashBin "start-develop-services" ''
                  ${pkgs.etcd}/bin/etcd \
                    --log-level warn \
                    --name .develop
                '')
                (writers.writeBashBin "watch-develop-clavelapi" ''
                  export PORT=8080
                  ${pkgs.watchexec}/bin/watchexec -e go -r "go run ./cmd/clavelapi"
                '')
                (writers.writeBashBin "watch-develop-clavelcontroller" ''
                  ${pkgs.watchexec}/bin/watchexec -e go -r "go run ./cmd/clavelcontroller"
                '')
                go
                buf
                protoc-gen-go
                protoc-gen-connect-go
                gopls
                watchexec
                etcd
              ];
            };
        };
      flake = {
        flakeModule = ./flake-module.nix;
        lib = import ./lib.nix { nixpkgs-lib = inputs.nixpkgs.lib; };
      };
    };
}
