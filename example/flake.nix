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
          nixosConfigurations.vmx = withSystem "x86_64-linux" (
            { pkgs, ... }:
            inputs.nixpkgs.lib.nixosSystem {
              specialArgs = {
                inherit inputs;
              };
              modules = [
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
                    networking.hostName = "nixos";
                    networking.useDHCP = true;
                    services.openssh.enable = true;
                    services.openssh.settings.PermitRootLogin = "yes";
                    users.users.root.password = "root";
                    services.xserver.enable = false;
                    system.stateVersion = "25.11";
                  }
                )
              ];
            }
          );
          clavelDeployments."vm1" = {
              template = inputs.clavel.lib.mkNixosDeploymentTemplate self.nixosConfigurations.vmx;
            };
          clavelUnits."vm1" = {
              deployment = self.clavelDeployments."vm1".name;
          };
        };
      }
    );
}
