# Update nixpkgs

[package.nix](https://github.com/NixOS/nixpkgs/blob/kubexporter/pkgs/by-name/ku/kubexporter/package.nix)

## git hash

```bash
nix-env -iA nixpkgs.nix-prefetch-git
nix-prefetch-git --url https://github.com/bakito/kubexporter --rev v0.6.4
```

## File

https://github.com/bakito/nixpkgs/tree/master/pkgs/by-name/ku/kubexporter

`pkgs/by-name/ku/kubexporter/package.nix`

### Changes

`  version = "0.6.4";`

## build

```bash
nix-build -A  kubexporter
```

## Test

```bash
./result/bin/kubexporter --version
```

## Commit

`kubexporter: 0.6.3 ->  0.6.4`

## nixpkgs-update



git clone https://github.com/nix-community/nixpkgs-update.git
cd nixpkgs-update



nix develop --extra-experimental-features nix-command --extra-experimental-features flakes


https://nix-community.github.io/nixpkgs-update/interactive-updates/#interactive-updates

git remote add upstream "https://github.com/NixOS/nixpkgs.git"
git fetch upstream
/home/bakito/git/nixpkgs-update/result/bin/nixpkgs-update update "kubexporter 0.6.3 0.6.4" --pr
