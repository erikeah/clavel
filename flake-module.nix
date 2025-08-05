{ lib, flake-parts-lib, ... }:
let
  inherit (lib) mkOption types;
  inherit (flake-parts-lib) mkSubmoduleOptions;
  clavel-lib = import ./lib.nix { nixpkgs-lib = lib; };
in
{
  options = {
    flake = mkSubmoduleOptions {
      clavelUnits = mkOption {
        type = types.lazyAttrsOf types.attrs;
        default = { };
        description = "Units with injected name field";
        apply = clavel-lib.injectName;
      };
      clavelDeployments = mkOption {
        type = types.lazyAttrsOf types.attrs;
        default = { };
        description = "Units with injected name field";
        apply = clavel-lib.injectName;
      };
    };
  };
}
