{ nixpkgs-lib, ... }:
let
  inherit (nixpkgs-lib) recursiveUpdate mapAttrs;
  inherit (builtins) toString;
in
{
  mkNixosUnit = attrs: {
    inherit (attrs) strategies profile;
    type = "nixos";
    storePath = toString attrs.configuration.config.system.build.toplevel;
  };
}
