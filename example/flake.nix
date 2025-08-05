{
  inputs = {
    flake-parts.url = "github:hercules-ci/flake-parts";
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    clavel.url = "path:../";
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
          inputs.flake-parts.flakeModules.flakeModules
          inputs.clavel.flakeModule
        ];
        systems = [
          "x86_64-linux"
          "aarch64-linux"
          "aarch64-darwin"
          "x86_64-darwin"
        ];
        flake = {
          nixosConfigurations.claveld = withSystem "x86_64-linux" (
            { pkgs, ... }:
            inputs.nixpkgs.lib.nixosSystem {
              specialArgs = {
                inherit inputs;
              };
              modules = [
                /*
                inputs.clavel.nixosModules.claveld
                inputs.clavel.nixosModules.clavel-nixos-agent
                */
                (
                  { config, pkgs, ... }:
                  {
                    nixpkgs.hostPlatform = "x86_64-linux";
                    boot.loader.grub.device = "/dev/vda";
                    boot.initrd.availableKernelModules = [
                      "virtio_pci"
                      "virtio_blk"
                      "virtio_net"
                    ];
                    boot.kernelModules = [ ];
                    fileSystems."/" = {
                      device = "/dev/vda1";
                      fsType = "ext4";
                    };
                    networking.hostName = "claveld";
                    networking.useDHCP = true;
                    services.openssh.enable = true;
                    services.openssh.settings.PermitRootLogin = "yes";
                    users.users.root.password = "root";
                    services.xserver.enable = false;
                    system.stateVersion = "25.11";
                  }
                )
                /*
                {
                  services.claveld.enable = true;
                  # Add clavel-nixos-agent to self manage
                  services.clavel-nixos-agent.enable = true;
                  services.clavel-nixos-agent.claveld.address = config.networking.hostName;
                  services.clavel-nixos-agent.units = [ "vm1" ];
                }
                */
              ];
            }
          );
          clavelUnits."claveld" = inputs.clavel.lib.mkNixosUnit {
            configuration = self.nixosConfigurations.claveld;
            profile = "/nix/var/nix/profiles/system";
            strategies = [
                { type = "local"; }
            ];
          };
        };
      }
    );
}
