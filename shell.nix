with (import (fetchTarball https://github.com/nixos/nixpkgs/archive/master.tar.gz) {});

let

 basePackages = [
  go
  ];

in mkShell {
  buildInputs = basePackages;

}
