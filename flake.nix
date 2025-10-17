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
                (writers.writeBashBin "go-modulepath-deps" ''
                  MODULEPATH=$1
                  MODNAME=$(go list -m)
                  go list -f '{{ join .Deps "\n" }}' $MODULEPATH | \
                  grep $MODNAME | \
                  sed "s|$MODNAME|.|"
                '')
                (writers.writeBashBin "develop-start-services" ''
                  ${pkgs.etcd}/bin/etcd \
                    --log-level warn \
                    --name .develop
                '')
                (writers.writeBashBin "develop-watch-clavelapi" ''
                  export PORT=8080
                  ${pkgs.watchexec}/bin/watchexec $(for path in $(go-modulepath-deps ./cmd/clavelapi); do printf -- ' -W %s' $path; done) -r "go run ./cmd/clavelapi"
                '')
                (writers.writeBashBin "develop-debug-clavelapi" ''
                  export PORT=8080
                  export CGO_CFLAGS="-O1"
                  ${pkgs.delve}/bin/dlv debug ./cmd/clavelapi
                '')
                (writers.writeBashBin "develop-watch-clavelcontroller" ''
                  ${pkgs.watchexec}/bin/watchexec $(for path in $(go-modulepath-deps ./cmd/clavelcontroller); do printf -- ' -W %s' $path; done) -r "go run ./cmd/clavelcontroller"
                '')
                buf
                coreutils
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
