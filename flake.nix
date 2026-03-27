{
  description = "wt - worktree manager with tmux session integration";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "wt";
          version = "0.1.0";
          src = ./.;
          vendorHash = null;

          ldflags = [
            "-s" "-w"
            "-X github.com/analogrelay/wt/cmd.Version=0.1.0"
          ];

          meta = with pkgs.lib; {
            description = "Worktree manager with tmux session integration";
            homepage = "https://github.com/analogrelay/wt";
            license = licenses.mit;
            mainProgram = "wt";
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            goreleaser
            golangci-lint
          ];
        };
      }
    );
}
