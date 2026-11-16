{
  description = "Tools installed into the numen runner images";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/f9a75f41be3a74eab55d3a7679d34ad6cc0da977";

  outputs =
    { nixpkgs, ... }:
    {
      packages = nixpkgs.lib.genAttrs [ "aarch64-darwin" ] (
        system:
        let
          pkgs = import nixpkgs {
            inherit system;
            config.allowUnfree = true;
          };
        in
        {
          default = pkgs.buildEnv {
            name = "numen-runner-tools";
            paths = [
              pkgs.nodejs_22
              pkgs.git
              pkgs.claude-code
              pkgs.codex
              pkgs.gemini-cli
              pkgs.yt-dlp
              pkgs.ffmpeg
            ];
          };
        }
      );
    };
}
