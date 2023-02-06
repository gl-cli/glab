# GLab

![GLab](docs/assets/glab-logo.png)

[![Go Report Card](https://goreportcard.com/badge/gitlab.com/gitlab-org/cli)](https://goreportcard.com/report/gitlab.com/gitlab-org/cli)
![Coverage](https://gitlab.com/gitlab-org/cli/badges/main/coverage.svg)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go#version-control)
[![Gitpod Ready-to-Code](https://img.shields.io/badge/Gitpod-Ready--to--Code-blue?style=flat&logo=gitpod&logoColor=white)](https://gitpod.io/#https://gitlab.com/gitlab-org/cli/-/tree/main/)

GLab is an open source GitLab CLI tool bringing GitLab to your terminal next to where you are already working with `git` and your code without switching between windows and browser tabs. Work with issues, merge requests, **watch running pipelines directly from your CLI** among other features.

`glab` is available for repositories hosted on GitLab.com and self-managed GitLab instances. `glab` supports multiple authenticated GitLab instances and automatically detects the authenticated hostname from the remotes available in the working Git directory.

![command example](docs/assets/glabgettingstarted.gif)

## Table of contents

- [Table of contents](#table-of-contents)
- [Usage](#usage)
- [Demo](#demo)
- [Documentation](#documentation)
- [Installation](#installation)
  - [Homebrew](#homebrew)
  - [Other installation methods](#other-installation-methods)
  - [Building from source](#building-from-source)
    - [Prerequisites for building from source](#prerequisites-for-building-from-source)
- [Authentication](#authentication)
- [Configuration](#configuration)
- [Environment variables](#environment-variables)
- [Issues](#issues)
- [Contributing](#contributing)

## Usage

To get started with `glab`:

1. Follow the [installation instructions](#installation) appropriate for your operating system.
1. [Authenticate](#authentication) into your instance of GitLab.
1. Optional. Configure `glab` further to meet your needs:
   - Set any needed global, per-project, or per-host [configuration](#configuration).
   - Set any needed [environment variables](#environment-variables).

You're ready! Run `glab --help` to view a list of core commands. Commands follow this pattern:

```shell
glab <command> <subcommand> [flags]
```

Many core commands also have sub-commands. Some examples:

- List merge requests assigned to you: `glab mr list --assignee=@me`
- List review requests for you: `glab mr list --reviewer=@me`
- Approve a merge request: `glab mr approve 235`
- Create an issue, and add milestone, title, and label: `glab issue create -m release-2.0.0 -t "My title here" --label important`

## Demo

[![asciicast](https://asciinema.org/a/368622.svg)](https://asciinema.org/a/368622)

## Documentation

Read the [documentation](https://gitlab.com/gitlab-org/cli/-/tree/main/docs/source) for usage instructions or check out `glab help`.

## Installation

Download a binary suitable for your OS at the [releases page](https://gitlab.com/gitlab-org/cli/-/releases).
Other installation methods depend on your operating system.

### Homebrew

Homebrew is the officially supported method for macOS, Linux, and Windows (through [Windows Subsystem for Linux](https://learn.microsoft.com/en-us/windows/wsl/install))

- Homebrew
  - Install with: `brew install glab`
  - Update with: `brew upgrade glab`

### Other installation methods

Other options to install the GitLab CLI that may not be officially support or are maintained by the community are [also available](docs/installation_options.md)

### Building from source

If a supported binary for your OS is not found at the [releases page](https://gitlab.com/gitlab-org/cli/-/releases), you can build from source:

#### Prerequisites for building from source

- `make`
- Go 1.18+

To build from source:

1. Run the command `go version` to verify that Go version 1.18 or later is installed.
   If `go` is not installed, follow instructions on [the Go website](https://go.dev/doc/install).
1. Clone this repository: `git clone https://gitlab.com/gitlab-org/cli.git glab`
1. Change into the project directory: `cd glab`
1. If you have `$GOPATH/bin` or `$GOBIN` in your `$PATH`, run `make install` to install in `$GOPATH/bin`).
1. If you do not have `$GOPATH/bin` or `$GOBIN` in your `$PATH`:
   1. Run `make` to build the project.
   1. Run `export PATH=$PWD/bin:$PATH` to update your PATH with the newly compiled project.
1. Run `glab version` to confirm that it worked.

## Authentication

To authenticate your installation of `glab`:

1. Get a GitLab personal access token with at least the `api`
   and `write_repository` scopes. Use the method appropriate for your instance:
   - For GitLab.com, create one at the [Personal access tokens](https://gitlab.com/-/profile/personal_access_tokens) page.
   - For self-managed instances, visit `https://gitlab.example.com/-/profile/personal_access_tokens`,
     modifying `gitlab.example.com` to match the domain name of your instance.
1. Start interactive setup: `glab auth login`
1. Authenticate with the method appropriate for your GitLab instance:
   - For GitLab SaaS, authenticate against `gitlab.com` by reading the token
     from a file: `glab auth login --stdin < myaccesstoken.txt`
   - For self-managed instances, authenticate by reading from a file:
     `glab auth login --hostname salsa.debian.org --stdin < myaccesstoken.txt`
   - Authenticate with token and hostname: `glab auth login --hostname gitlab.example.org --token xxxxx`
     Not recommended for shared environments.

## Configuration

By default, `glab` follows the
[XDG Base Directory Spec](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html).
Configure it globally, locally, or per-host:

- **Globally**: run `glab config set --global editor vim`.
  - The global configuration file is available at `~/.config/glab-cli`.
  - To override this location, set the `GLAB_CONFIG_DIR` environment variable.
- **The current directory**: run `glab config set editor vim` in any folder in a Git repository.
  - The local configuration file is available at `.git/glab-cli` in the current working Git directory.
- **Per host**: run `glab config set editor vim --host gitlab.example.org`, changing
  the `--host` parameter to meet your needs.
  - Per-host configuration info is always stored in the global configuration file, with or without the `global` flag.

## Environment variables

- `GITLAB_TOKEN`: an authentication token for API requests. Setting this avoids being
  prompted to authenticate and overrides any previously stored credentials.
  Can be set in the config with `glab config set token xxxxxx`
- `GITLAB_URI` or `GITLAB_HOST`: specify the URL of the GitLab server if self-managed (eg: `https://gitlab.example.com`). Default is `https://gitlab.com`.
- `GITLAB_API_HOST`: specify the host where the API endpoint is found. Useful when there are separate (sub)domains or hosts for Git and the API endpoint: defaults to the hostname found in the Git URL
- `GITLAB_REPO`: Default GitLab repository used for commands accepting the `--repo` option. Only used if no `--repo` option is given.
- `GITLAB_GROUP`: Default GitLab group used for listing merge requests, issues and variables. Only used if no `--group` option is given.
- `REMOTE_ALIAS` or `GIT_REMOTE_URL_VAR`: `git remote` variable or alias that contains the GitLab URL.
  Can be set in the config with `glab config set remote_alias origin`
- `VISUAL`, `EDITOR` (in order of precedence): the editor tool to use for authoring text.
  Can be set in the config with `glab config set editor vim`
- `BROWSER`: the web browser to use for opening links.
   Can be set in the configuration with `glab config set browser mybrowser`
- `GLAMOUR_STYLE`: environment variable to set your desired Markdown renderer style
  Available options are (`dark`|`light`|`notty`) or set a [custom style](https://github.com/charmbracelet/glamour#styles)
- `NO_COLOR`: set to any value to avoid printing ANSI escape sequences for color output.
- `FORCE_HYPERLINKS`: set to `1` to force hyperlinks to be output, even when not outputting to a TTY

## Issues

If you have an issue: report it on the [issue tracker](https://gitlab.com/gitlab-org/cli/-/issues)

## Contributing

Feel like contributing? That's awesome! We have a [contributing guide](https://gitlab.com/gitlab-org/cli/-/blob/main/CONTRIBUTING.md) and [Code of conduct](https://gitlab.com/gitlab-org/cli/-/blob/main/CODE_OF_CONDUCT.md) to help guide you.
