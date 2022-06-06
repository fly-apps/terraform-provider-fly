with (import (fetchTarball https://github.com/nixos/nixpkgs/archive/592dc9ed7f049c565e9d7c04a4907e57ae17e2d9.tar.gz) {});

let

 basePackages = [
  go
  ];

in mkShell {
  buildInputs = basePackages;

}
