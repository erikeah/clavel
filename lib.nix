{ nixpkgs-lib, ... }:
let
  inherit (nixpkgs-lib) recursiveUpdate mapAttrs;
  inherit (builtins) toString;
in
{
  injectName = attrs: mapAttrs (key: value: recursiveUpdate value { name = key; }) attrs;
  mkNixosUnit = attrs: {
    inherit (attrs) access;
    type = "nixos";
    storePath = toString attrs.configuration.config.system.build.toplevel;
  };
}
