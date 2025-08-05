{ nixpkgs-lib, ... }:
let
  inherit (nixpkgs-lib) recursiveUpdate mapAttrs;
  inherit (builtins) toString;
in
{
  injectName = attrSet: mapAttrs (key: value: recursiveUpdate value { name = key; }) attrSet;
  mkNixosDeploymentTemplate = nixosConfiguration: {
    type = "nixos";
    storePath = toString nixosConfiguration.config.system.build.toplevel;
  };
}
