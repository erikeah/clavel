{ ... }:
{
  # Transform an evaluation flake reference into the NixOS toplevel path, e.g.
  # github:user/config#host -> github:user/config#nixosConfigurations.host.config.system.build.toplevel
  reference = evaluationRef:
    let
      parts = builtins.match "(.*)#(.*)" evaluationRef;
      source = builtins.elemAt parts 0;
      target = builtins.elemAt parts 1;
    in
    "${source}#nixosConfigurations.${target}.config.system.build.toplevel";

  # Produce a deployment action for a NixosUnit resource.
  apply = specJson: evaluationRef: derivation:
    let
      spec = builtins.fromJSON specJson;
    in
    {
      type = "command";
      command = [
        "nix-env"
        "--profile"
        (if spec ? profile then spec.profile else "/nix/var/nix/profiles/system")
        "--install"
        derivation
      ];
    };
}