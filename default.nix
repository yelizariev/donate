let
  pkgs = import <nixpkgs> {};
in with pkgs; buildGoPackage rec {
  pname = "donate";
  version = "master";

  buildInputs = [ makeWrapper ];

  goPackagePath = "code.dumpstack.io/tools/${pname}";

  src = ./.;
  goDeps = ./deps.nix;

  # TODO
  # The problem here is that one dependency relies on the native
  # library so, build will be stopped because RPATH will contain
  # /build/go/...
  preFixup = let libPath = stdenv.lib.makeLibraryPath [ glibc ]; in ''
    RUNTIME=/build/go/src/github.com/wasmerio/go-ext-wasm/wasmer/libwasmer_runtime_c_api.so
    mkdir $bin/lib
    cp $RUNTIME $bin/lib/
    patchelf --set-rpath $bin/lib:${libPath} "$bin/bin/${pname}"
    patchelf --set-rpath $bin/lib:${libPath} "$bin/bin/${pname}-ci"
  '';

  postFixup = ''
    wrapProgram $bin/bin/${pname} \
      --prefix PATH : "${lib.makeBinPath [ which electrum ]}"
  '';
}
