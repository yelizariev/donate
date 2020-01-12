let
  pkgs = import <nixpkgs> {};
in with pkgs; buildGoPackage rec {
  pname = "donate";
  version = "master";

  buildInputs = [ makeWrapper ];

  goPackagePath = "code.dumpstack.io/tools/${pname}";

  src = ./.;
  goDeps = ./deps.nix;

  postFixup = ''
    wrapProgram $bin/bin/${pname} \
      --prefix PATH : "${lib.makeBinPath [ which electrum ]}"
  '';
}
