{
  inputs = {
    flake-parts.url = "github:hercules-ci/flake-parts";
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    clavel.url = "./..";
  };

  outputs =
    inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } (
      {
        self,
        withSystem,
        config,
        ...
      }:
      {
        imports = [
          inputs.clavel.flakeModule
        ];
        systems = [
          "x86_64-linux"
          "aarch64-linux"
          "aarch64-darwin"
          "x86_64-darwin"
        ];
        flake = {
          nixosConfigurations.system = inputs.nixpkgs.lib.nixosSystem {
              modules = [];
          };
          clavelConfigurations.default = inputs.clavel.lib.clavelDefinition {
            imports = [
              inputs.clavel.clavelModules.nixos # This allows nixosUnits to be defined by automatically creating crd and module definition
            ];
            nixosUnits."system" = {
              name = "system"; # Optional; automatically taken from attr name
              # This creates the evaluation and configure system nixos unit to set evaluationRef to "system"
              evaluation = {
                name = "system"; # Optional; automatically inherit
                /*
                    Because this is the nixos unit
                    integration reference will be transformed to
                    github:erikeah/clavel?dir=example#nixosConfigurations.system.config.system.build.toplevel
                */
                reference = "github:erikeah/clavel?dir=example#system";
              };
            };
          };
        };
      }
    );
}
