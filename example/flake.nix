{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    clavel.url = "./..";
  };

  outputs = inputs: {
    # Non-functional nixos system, is just for illustration.
    nixosConfigurations.server = inputs.nixpkgs.lib.nixosSystem { };
    clavelConfigurations.default = inputs.clavel.lib.clavelDefinition {
      imports = [
        inputs.clavel.clavelModules.nixos # This allows nixosUnits to be defined by automatically creating crd and module definition
      ];
      nixosUnits."server" = {
        name = "server"; # Optional; automatically taken from attr name
        # This creates the evaluation and configure server nixos unit to set evaluationRef to "server"
        evaluation = {
          name = "server"; # Optional; automatically inherit
          /*
              Because this is the nixos unit
              integration reference will be transformed to
              github:erikeah/clavel?dir=example#nixosConfigurations.server.config.server.build.toplevel
          */
          reference = "github:erikeah/clavel?dir=example#server";
        };
      };
    };
  };
}
