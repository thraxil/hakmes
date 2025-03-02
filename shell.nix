{ pkgs ? import (fetchTarball "https://github.com/NixOS/nixpkgs/archive/refs/tags/25.05-pre.tar.gz") {} }:

pkgs.mkShell {
  buildInputs = [
    pkgs.go
    pkgs.gcc
    pkgs.libcap
  ];

  shellHook = ''
  '';
  HAKMES_PORT="9300";
  HAKMES_CASK_BASE="http://localhost:9201";
  HAKMES_CHUNK_SIZE="16777216";
  HAKMES_DB_PATH="hakmes.db";
}
