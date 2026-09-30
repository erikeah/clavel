{ nixpkgs-lib, ... }:
let
  inherit (nixpkgs-lib) attrValues evalModules mkOption types;
  inherit (builtins) removeAttrs;

  crdModule = {
    options = {
      group = mkOption { type = types.str; };
      version = mkOption { type = types.str; };
      kind = mkOption { type = types.str; };
      plural = mkOption { type = types.str; };
      schema = mkOption { type = types.str; };
      specMessage = mkOption { type = types.str; };
      actionModule = mkOption { type = types.str; default = ""; };
      metadata = mkOption { type = types.attrs; default = { }; };
    };
  };

  resourceModule = {
    options = {
      group = mkOption { type = types.str; };
      version = mkOption { type = types.str; };
      plural = mkOption { type = types.str; };
      kind = mkOption { type = types.str; };
      name = mkOption { type = types.str; };
      metadata = mkOption { type = types.attrs; default = { }; };
      # protojson-encoded spec, carried as a JSON string (Resource.spec is bytes).
      spec = mkOption { type = types.str; };
    };
  };

  evaluationModule = {
    options = {
      name = mkOption { type = types.str; };
      metadata = mkOption { type = types.attrs; default = { }; };
      spec = mkOption { type = types.attrs; };
    };
  };

  baseModule = {
    options = {
      clavel.customResourceDefinitions = mkOption {
        type = types.lazyAttrsOf (types.submodule crdModule);
        default = { };
        description = "Custom resource definitions contributed by clavel modules.";
      };
      clavel.resources = mkOption {
        type = types.lazyAttrsOf (types.submodule resourceModule);
        default = { };
        description = "Custom resource instances derived from declared configurations.";
      };
      clavel.evaluations = mkOption {
        type = types.lazyAttrsOf (types.submodule evaluationModule);
        default = { };
        description = "Evaluations referenced by declared configurations.";
      };
    };
  };
in
{
  # Runs a clavel module system over `imports` and the declared <kind>s.*
  # attrs, producing the apply descriptor (CRDs, resources, evaluations)
  # consumed by `clavelctl apply`.
  clavelDefinition = args@{ imports ? [ ], ... }:
    let
      declared = removeAttrs args [ "imports" ];
      userModule = {
        config.clavel = declared;
      };
      evaluated = evalModules {
        modules = [ baseModule userModule ] ++ imports;
      };
      cfg = evaluated.config.clavel;
    in
    {
      customResourceDefinitions = attrValues cfg.customResourceDefinitions;
      resources = attrValues cfg.resources;
      evaluations = attrValues cfg.evaluations;
    };

  mkNixosUnit = attrs: {
    inherit (attrs) strategies profile;
    type = "nixos";
    storePath = toString attrs.configuration.config.system.build.toplevel;
  };
}