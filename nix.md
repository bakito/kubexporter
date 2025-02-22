# Update nixpkgs

[package.nix](https://github.com/NixOS/nixpkgs/blob/kubexporter/pkgs/by-name/ku/kubexporter/package.nix)

## git hash

```bash
nix-env -iA nixpkgs.nix-prefetch-git
nix-prefetch-git --url https://github.com/bakito/kubexporter --rev v0.6.2
```

## build

```bash
nix-build -A  kubexporter
```

## Test

```bash
./result/bin/kubexporter 
```

