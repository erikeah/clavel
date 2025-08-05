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
                go
                buf
                protoc-gen-go
                protoc-gen-connect-go
                gopls
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
