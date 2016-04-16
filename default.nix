with import <nixpkgs> {}; {
  defaultEnv = stdenv.mkDerivation {
    name = "default";
    buildInputs = [
	portmidi
	portaudio
    ];
  };
}
