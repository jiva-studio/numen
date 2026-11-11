# What a Nix machine installs, and the tools a contributor builds with. The
# package compiles nothing: it takes the Linux build that was published and
# links it against the toolkit in the store.
{
  description = "numen — notes, a hierarchy with several parents, and the sources under them";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  inputs.nixpkgs-node.url = "github:NixOS/nixpkgs/f9a75f41be3a74eab55d3a7679d34ad6cc0da977";

  outputs =
    { self, nixpkgs, nixpkgs-node }:
    let
      lib = nixpkgs.lib;
      systems = [ "x86_64-linux" ];
      devSystems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
      ];

      # numen is not free software, and a package set says so before it builds
      # one. This is the set this flake builds its own package out of; a machine
      # taking the overlay answers for that on its own.
      on =
        f:
        lib.genAttrs systems (
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
      assets = ./apps/desktop/build/linux;

      readLines = file: lib.splitString "\n" (builtins.readFile file);

      # The version after `prefix` on the first line that starts with it.
      findVersion =
        file: prefix:
        lib.removePrefix prefix (lib.findFirst (lib.hasPrefix prefix) "" (map lib.trim (readLines file)));

      # The versions CI runs. Go and Node are read from go.work and
      # .node-version, the Go plugins from the tool block of libs/core/protocol/go.mod;
      # golangci-lint and buf are also named in .github/workflows, and
      # golangci-lint in tools/checks/lib/golangci.mjs.
      versions = {
        go = findVersion ./go.work "go ";
        node = lib.trim (builtins.readFile ./.node-version);
        golangci-lint = "2.14.0";
        buf = "1.72.0";
        protoc-gen-go = findVersion ./libs/core/protocol/go.mod "google.golang.org/protobuf v";
        protoc-gen-connect-go = findVersion ./libs/core/protocol/go.mod "connectrpc.com/connect v";
      };

      # Fails evaluation when a tool is not at the version CI runs.
      requireVersion =
        name: package:
        lib.throwIfNot (package.version == versions.${name})
          "flake.nix: ${name} is ${package.version}, the repository asks for ${versions.${name}}"
          package;

      getTools =
        pkgs:
        let
          go = pkgs.go_1_26.overrideAttrs (
            finalAttrs: _: {
              version = "1.26.11";
              src = pkgs.fetchurl {
                url = "https://go.dev/dl/go${finalAttrs.version}.src.tar.gz";
                hash = "sha256-1EEBsP8n7SpMda1GwXucdDWSCeLrLKfJ32y0/baOGWk=";
              };
            }
          );
          golangci-lint = pkgs.golangci-lint.overrideAttrs (
            finalAttrs: _: {
              version = "2.14.0";
              src = pkgs.fetchFromGitHub {
                owner = "golangci";
                repo = "golangci-lint";
                tag = "v${finalAttrs.version}";
                hash = "sha256-HATA7JKHwEouM+8jYZbQrkX7p4gut4IpyTvcBexu/4o=";
              };
              vendorHash = "sha256-ekP/zDhYMpMG+tYAyYHNfLOCt/JkxrXUw1jHUEfsM8k=";
            }
          );
          protoc-gen-connect-go = pkgs.protoc-gen-connect-go.overrideAttrs (
            finalAttrs: _: {
              version = "1.21.1";
              src = pkgs.fetchFromGitHub {
                owner = "connectrpc";
                repo = "connect-go";
                tag = "v${finalAttrs.version}";
                hash = "sha256-UM8C68NhhsfASxfn/RWh1mFWfngcdY8D/gbzRFbOlk0=";
              };
              vendorHash = "sha256-0XmH9V7Bbuzj//87Ev6KRy6ijsgh7K6JRhhM5WY7K38=";
              doCheck = false;
            }
          );
        in
        lib.mapAttrsToList requireVersion {
          inherit go golangci-lint protoc-gen-connect-go;
          node = nixpkgs-node.legacyPackages.${pkgs.stdenv.hostPlatform.system}.nodejs_22;
          inherit (pkgs) buf protoc-gen-go;
        };
    in
    {
      packages = on (pkgs: rec {
        numen = pkgs.callPackage ./tools/build-nix/numen.nix { inherit assets; };
        default = numen;
      });

      overlays.default = final: _prev: {
        numen = final.callPackage ./tools/build-nix/numen.nix { inherit assets; };
      };

      # protoc-gen-es is the one in node_modules, at the version of package-lock.json;
      # libs/core/protocol/buf.gen.yaml runs it from there. The flake a release hands
      # out (release.yml) carries no go.work, and so no devShell.
      devShells = lib.optionalAttrs (builtins.pathExists ./go.work) (
        lib.genAttrs devSystems (
          system:
          let
            pkgs = nixpkgs.legacyPackages.${system};
          in
          {
            default = pkgs.mkShell {
              packages = getTools pkgs;
              GOTOOLCHAIN = "local";
            };
          }
        )
      );

      formatter = on (pkgs: pkgs.nixfmt-rfc-style);
    };
}
