{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = with pkgs; [
    go
    gopls
  ];

  shellHook = ''
    export GOPATH="$HOME/go"
    export PATH="$GOPATH/bin:$PATH"
  '';
}
