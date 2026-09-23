# What a Nix machine installs. Nothing is compiled here: this takes the Linux
# build that was published and links it against the toolkit in the store.
{
  description = "numen — notes, a hierarchy with several parents, and the sources under them";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" ];

      # numen is not free software, and a package set says so before it builds
      # one. This is the set this flake builds its own package out of; a machine
      # taking the overlay answers for that on its own.
      on =
        f:
        nixpkgs.lib.genAttrs systems (
          system:
          f (
            import nixpkgs {
              inherit system;
              config.allowUnfree = true;
            }
          )
        );

      # The desktop entries and the icons come from this tree; only the two
      # binaries are fetched.
      assets = ./modules/apps/desktop/build/linux;
    in
    {
      packages = on (pkgs: rec {
        numen = pkgs.callPackage ./modules/tools/nix/numen.nix { inherit assets; };
        default = numen;
      });

      overlays.default = final: _prev: {
        numen = final.callPackage ./modules/tools/nix/numen.nix { inherit assets; };
      };

      formatter = on (pkgs: pkgs.nixfmt-rfc-style);
    };
}
