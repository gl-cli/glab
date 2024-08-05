---
stage: Create
group: Code Review
info: To determine the technical writer assigned to the Stage/Group associated with
this page, see https://about.gitlab.com/handbook/product/ux/technical-writing/#assignments
---

# Other options to install the GitLab CLI

These installation instructions are either not officially supported by GitLab, or
are maintained by the community.

## Table of contents

- [macOS](#macos)
- [Windows](#windows)
- [Linux](#linux)
  - [Homebrew](#homebrew)
  - [ASDF](#asdf)
  - [Snapcraft (currently out of date)](#snapcraft)
  - [Arch Linux](#arch-linux)
  - [Alpine Linux](#alpine-linux)
    - [Install a pinned version from edge](#install-a-pinned-version-from-edge)
    - [Alpine Linux Docker-way](#alpine-linux-docker-way)
  - [Fedora](#fedora)
  - [Nix/NixOS](#nixnixos)
  - [MPR (Debian/Ubuntu)](#mpr-debianubuntu)
    - [Prebuilt-MPR](#prebuilt-mpr)
  - [Spack](#spack)
- [Docker](#docker)
  - [GitLab CICD](#gitlab-cicd)

## macOS

- Homebrew (officially supported)
  - Install with: `brew install glab`
  - Update with: `brew upgrade glab`
- [MacPorts](https://ports.macports.org/port/glab/details/):
  - Install with: `sudo port install glab`
  - Update with: `sudo port selfupdate && sudo port upgrade glab`
- [ASDF tool version manager](https://asdf-vm.com/guide/introduction.html):
  - Install with: `asdf plugin add glab; asdf install glab latest; asdf global glab latest`
- Install into `usr/bin` with a shell script:
  `curl -s "https://gitlab.com/gitlab-org/cli/-/raw/main/scripts/install.sh" | sudo sh`

  Before running any install script, review its contents.

## Windows

- Homebrew (through [Windows Subsystem for Linux](https://learn.microsoft.com/en-us/windows/wsl/install)) (officially supported)
  - Install with: `brew install glab`
  - Update with: `brew upgrade glab`
- [WinGet](https://github.com/microsoft/winget-cli)
  - Install with: `winget install glab.glab`
  - Update with: `winget install glab.glab`
- [Chocolatey](https://chocolatey.org)
  - Install with: `choco install glab`
  - Update with: `choco upgrade glab`
- [scoop](https://scoop.sh)
  - Install with: `scoop install glab`
  - Update with: `scoop update glab`
- [ASDF tool version manager](https://asdf-vm.com/guide/introduction.html):
  - Requires [Windows Subsystem for Linux (WSL)](https://learn.microsoft.com/en-us/windows/wsl/install).
  - Install with: `asdf plugin add glab; asdf install glab latest; asdf global glab latest`
- Download an EXE installer or the `glab.exe` binary from the [releases page](https://gitlab.com/gitlab-org/cli/-/releases)

## Linux

- Download prebuilt binaries from the [releases page](https://gitlab.com/gitlab-org/cli/-/releases)

### Homebrew

Installing from Homebrew is the officially supported installation method for Linux.

- Install with: `brew install glab`
- Update with: `brew upgrade glab`

### ASDF

To install with the [ASDF tool version manager](https://asdf-vm.com/guide/introduction.html), run these commands:

```shell
asdf plugin add glab; asdf install glab latest; asdf global glab latest
```

### Snapcraft

This method is out of date. See [issue 1127](https://gitlab.com/gitlab-org/cli/-/issues/1127) for more information.

To install `glab` from the [Snap Store](https://snapcraft.io/glab):

1. Make sure you have [snap installed](https://snapcraft.io/docs/installing-snapd) on your Linux distribution.
1. Install the package: `sudo snap install --edge glab`
1. Grant `glab` access to SSH keys: `sudo snap connect glab:ssh-keys`

[![Download from the Snap Store](https://snapcraft.io/static/images/badges/en/snap-store-black.svg)](https://snapcraft.io/glab)

### Arch Linux

For Arch Linux, `glab` is available:

- From the [`extra/glab`](https://archlinux.org/packages/extra/x86_64/glab/) package.
- By downloading and installing an archive from the
  [releases page](https://gitlab.com/gitlab-org/cli/-/releases).
- From the [Snap Store](https://snapcraft.io/glab), if
  [snap](https://snapcraft.io/docs/installing-snap-on-arch-linux) is installed.
- Installing with the package manager: `pacman -S glab`

### Alpine Linux

`glab` is available on the [Alpine Community Repository](https://git.alpinelinux.org/aports/tree/community/glab?h=master) as `glab`.

When installing, use `--no-cache` so no `apk update` is required:

```shell
apk add --no-cache glab
```

#### Install a pinned version from edge

To ensure that by default edge is used to get the latest updates. We need the edge repository in `/etc/apk/repositories`.

Afterwards you can install it with `apk add --no-cache glab@edge`

We use `--no-cache` so an `apk update` is not required.

```shell
echo "@edge http://dl-cdn.alpinelinux.org/alpine/edge/community" >> /etc/apk/repositories
apk add --no-cache glab@edge
```

#### Alpine Linux Docker-way

Use edge directly

```shell
FROM alpine:3.13
RUN apk add --no-cache glab
```

Fetching latest glab version from edge

```shell
FROM alpine:3.13
RUN echo "@edge http://dl-cdn.alpinelinux.org/alpine/edge/community" >> /etc/apk/repositories
RUN apk add --no-cache glab@edge
```

### Fedora

Fedora users can find `glab` as `glab` in the official repositories.

Install it with the command `dnf install glab`.

### Nix/NixOS

Nix (NixOS) users can install from [nixpkgs](https://search.nixos.org/packages?channel=unstable&show=glab&from=0&size=30&sort=relevance&query=glab) with the command `nix-env -iA nixos.glab`.

### MPR (Debian/Ubuntu)

`glab` is available inside the [makedeb package repository](https://mpr.makedeb.org/packages/glab). To install, run the following:

```shell
git clone 'https://mpr.makedeb.org/glab'
cd glab/
makedeb -si
```

### WakeMeOps (Debian/Ubuntu)

`glab` also exists in the [WakeMeOps repository](https://docs.wakemeops.com/packages/glab/):

```shell
# Add WakeMeOps repository
curl -sSL "https://raw.githubusercontent.com/upciti/wakemeops/main/assets/install_repository" | sudo bash

# Install glab
sudo apt install glab
```

#### Prebuilt-MPR

The above method downloads glab from source and builds it before packaging it into a `.deb` package. If you don't want to compile or just want a prebuilt package, you can also install glab from the Prebuilt-MPR:

1. Set up [the Prebuilt-MPR on your system](https://docs.makedeb.org/prebuilt-mpr/getting-started/#setting-up-the-repository).
1. Install with the command `sudo apt install glab`.

#### Spack

- To install: `spack install glab`.
- To update: `spack uninstall glab && spack install glab`

## Docker

A Docker image for `glab` is available at
[`gitlab/glab`](https://hub.docker.com/r/gitlab/glab/):

```shell
docker pull gitlab/glab
```

### GitLab CI/CD

To use `glab` in a CI/CD pipeline, you must set the `entrypoint` of the image to
the `glab` executable itself. For more information, see the GitLab documentation
for [Override the entrypoint of an image](https://docs.gitlab.com/ee/ci/docker/using_docker_images.html#override-the-entrypoint-of-an-image). An example `.gitlab-ci.yml`:

Example `.gitlab-ci.yml`:

```yaml
example:
  stage: test
  image:
    name: "gitlab/glab"
    entrypoint: [""]
  script:
    - glab --version
```
