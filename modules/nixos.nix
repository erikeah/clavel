{ lib, config, ... }:
let
  inherit (lib) mapAttrs' mkOption nameValuePair types;
  generated = import ../nix/generated/nixos.nix { inherit lib; };
  resolve = key: unit:
    let
      resourceName = if unit.name == null then key else unit.name;
    in
    {
      inherit resourceName;
      evaluationName = if unit.evaluation.name == null then resourceName else unit.evaluation.name;
    };
  toSpec = key: unit:
    let
      resolved = resolve key unit;
    in
    {
      evaluation = resolved.evaluationName;
      profile = if unit.profile == null then "" else unit.profile;
      strategies = unit.strategies;
    };
  toResource = name: spec:
    {
      group = generated.descriptor.group;
      version = generated.descriptor.version;
      plural = generated.descriptor.plural;
      kind = generated.descriptor.kind;
      name = name;
      metadata = { };
      spec = builtins.toJSON spec;
    };
  toEvaluation = { unit, evaluationName }:
    {
      name = evaluationName;
      metadata = { };
      spec = {
        reference = unit.evaluation.reference;
      };
    };
in
{
  options.clavel.nixosUnits = mkOption {
    description = "NixOS units: deploy a NixOS configuration referenced by an Evaluation.";
    type = types.lazyAttrsOf (types.submodule {
      options = {
        name = mkOption {
          type = types.nullOr types.str;
          default = null;
          description = "Resource name; defaults to the attribute name.";
        };
        evaluation = mkOption {
          description = "Evaluation used to obtain the NixOS toplevel derivation.";
          type = types.submodule {
            options = {
              name = mkOption {
                type = types.nullOr types.str;
                default = null;
                description = "Evaluation name; defaults to the resource name.";
              };
              reference = mkOption {
                type = types.str;
                description = "Flake reference of the NixOS configuration, e.g. github:user/config#host.";
              };
            };
          };
        };
        profile = mkOption {
          type = types.nullOr types.str;
          default = null;
          description = "System profile to activate; defaults to the empty profile.";
        };
        strategies = mkOption {
          type = types.listOf (types.submodule {
            options = {
              type = mkOption { type = types.str; };
              host = mkOption { type = types.str; default = ""; };
            };
          });
          default = [ ];
        };
      };
    });
    default = { };
  };

  # Typed unit specs, validated against the generated proto-derived schema.
  options.clavel.nixosUnitSpecs = mkOption {
    description = "NixOS unit specs validated against the NixosUnitSpec schema.";
    type = types.lazyAttrsOf generated.spec;
    default = { };
  };

  config = {
    clavel.nixosUnitSpecs = mapAttrs' (key: unit: nameValuePair (resolve key unit).resourceName (toSpec key unit)) config.clavel.nixosUnits;
    clavel.customResourceDefinitions.nixos = generated.descriptor // {
      actionModule = "github:erikeah/clavel#clavelActions.nixos";
    };
    clavel.resources = mapAttrs' (name: spec: nameValuePair name (toResource name spec)) config.clavel.nixosUnitSpecs;
    clavel.evaluations = mapAttrs' (
      key: unit:
      let
        resolved = resolve key unit;
      in
      nameValuePair resolved.evaluationName (toEvaluation {
        inherit unit;
        evaluationName = resolved.evaluationName;
      })
    ) config.clavel.nixosUnits;
  };
}