# GLab

![GLab](docs/assets/glab-logo.png)

GLab is an open source GitLab CLI tool. It brings GitLab to your terminal, next to where you are already working with `git` and your code, without switching between windows and browser tabs. While it's powerful for issues and merge requests, `glab` does even more:

- View, manage, and retry CI/CD pipelines directly from your CLI.
- Create changelogs.
- Create and manage releases.
- Ask GitLab Duo Chat questions about Git.
- Manage GitLab agents for Kubernetes.

`glab` is available for repositories hosted on GitLab.com, GitLab Dedicated, and GitLab Self-Managed. It supports multiple authenticated GitLab instances, and automatically detects the authenticated hostname from the remotes available in your working Git directory.

![command example](docs/assets/glabgettingstarted.gif)

## Table of contents

- [Table of contents](#table-of-contents)
- [Requirements](#requirements)
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
- [Inspiration](#inspiration)

## Requirements

`glab` officially supports GitLab versions 16.0 and later. Certain commands might require
more recent versions. While many commands might work properly in GitLab versions
15.x and earlier, no support is provided for these versions.

## Usage

To get started with `glab`:

1. Follow the [installation instructions](#installation) appropriate for your operating system.
1. [Authenticate](#authentication) into your instance of GitLab.
1. Optional. Configure `glab` further to meet your needs:
   - 1Password users can configure it to [authenticate to `glab`](https://developer.1password.com/docs/cli/shell-plugins/gitlab/).
   - Set any needed global, per-project, or per-host [configuration](#configuration).
   - Set any needed [environment variables](#environment-variables).

You're ready!

### Core commands

Run `glab --help` to view a list of core commands in your terminal.

- [`glab alias`](docs/source/alias)
- [`glab api`](docs/source/api)
- [`glab auth`](docs/source/auth)
- [`glab changelog`](docs/source/changelog)
- [`glab check-update`](docs/source/check-update)
- [`glab ci`](docs/source/ci)
- [`glab cluster`](docs/source/cluster)
- [`glab completion`](docs/source/completion)
- [`glab config`](docs/source/config)
- [`glab duo`](docs/source/duo)
- [`glab incident`](docs/source/incident)
- [`glab issue`](docs/source/issue)
- [`glab label`](docs/source/label)
- [`glab mr`](docs/source/mr)
- [`glab release`](docs/source/release)
- [`glab repo`](docs/source/repo)
- [`glab schedule`](docs/source/schedule)
- [`glab securefile`](docs/source/securefile)
- [`glab snippet`](docs/source/snippet)
- [`glab ssh-key`](docs/source/ssh-key)
- [`glab stack`](docs/source/stack)
- [`glab user`](docs/source/user)
- [`glab variable`](docs/source/variable)

Commands follow this pattern:

```shell
glab <command> <subcommand> [flags]
```

Many core commands also have sub-commands. Some examples:

- List merge requests assigned to you: `glab mr list --assignee=@me`
- List review requests for you: `glab mr list --reviewer=@me`
- Approve a merge request: `glab mr approve 235`
- Create an issue, and add milestone, title, and label: `glab issue create -m release-2.0.0 -t "My title here" --label important`

### GitLab Duo for the CLI

The GitLab CLI also provides support for GitLab Duo AI/ML powered features. These include:

- [`glab duo ask`](docs/source/duo/ask.md)

Use `glab duo ask` to ask questions about `git` commands. It can help you remember a
command you forgot, or provide suggestions on how to run commands to perform other tasks.

## Demo

[![asciicast](https://asciinema.org/a/368622.svg)](https://asciinema.org/a/368622)

## Documentation

Read the [documentation](docs/source/index.md) for usage instructions or check out `glab help`.

## Installation

Download a binary suitable for your OS at the [releases page](https://gitlab.com/gitlab-org/cli/-/releases).
Other installation methods depend on your operating system.

### Homebrew

Homebrew is the officially supported package manager for macOS, Linux, and Windows (through [Windows Subsystem for Linux](https://learn.microsoft.com/en-us/windows/wsl/install))

- Homebrew
  - Install with: `brew install glab`
  - Update with: `brew upgrade glab`

### Other installation methods

Other options to install the GitLab CLI that may not be officially supported or are maintained by the community are [also available](docs/installation_options.md).

### Building from source

If a supported binary for your OS is not found at the [releases page](https://gitlab.com/gitlab-org/cli/-/releases), you can build from source:

#### Prerequisites for building from source

- `make`
- Go 1.22+

To build from source:

1. Run the command `go version` to verify that Go version 1.22 or later is installed.
   If `go` is not installed, follow instructions on [the Go website](https://go.dev/doc/install).
1. Run the `go install gitlab.com/gitlab-org/cli/cmd/glab@main` to install `glab` cmd in `$GOPATH/bin`.
1. The sources of `glab` will be in `$GOPATH/src/gitlab.com/gitlab-org/cli`.
1. If you do not have `$GOPATH/bin` or `$GOBIN` in your `$PATH`, run `export PATH=$PWD/bin:$PATH`
   to update your PATH with the newly compiled project.
1. Run `glab version` to confirm that it worked.

## Authentication

### OAuth (GitLab.com)

To authenticate your installation of `glab` with an OAuth application connected to GitLab.com:

1. Start interactive setup with `glab auth login`.
1. For the GitLab instance you want to sign in to, select **GitLab.com**.
1. For the login method, select **Web**. This selection launches your web browser
   to request authorization for the GitLab CLI to use your GitLab.com account.
1. Select **Authorize**.
1. Complete the authentication process in your terminal, selecting the appropriate options for your needs.

### OAuth (GitLab Self-Managed, GitLab Dedicated)

Prerequisites:

- You've created an OAuth application at the user, group, or instance level, and you
  have its application ID. For instructions, see how to configure GitLab
  [as an OAuth 2.0 authentication identity provider](https://docs.gitlab.com/integration/oauth_provider/)
  in the GitLab documentation.
- Your OAuth application is configured with these parameters:
  - **Redirect URI** is `http://localhost:7171/auth/redirect`.
  - **Confidential** is not selected.
  - **Scopes** are `openid`, `profile`, `read_user`, `write_repository`, and `api`.

To authenticate your installation of `glab` with an OAuth application connected
to your GitLab Self-Managed or GitLab Dedicated instance:

1. Store the application ID with `glab config set client_id <CLIENT_ID> --host <HOSTNAME>`.
   For `<CLIENT_ID>`, provide your application ID.
1. Start interactive setup with `glab auth login --hostname <HOSTNAME>`.
1. For the login method, select **Web**. This selection launches your web browser
   to request authorization for the GitLab CLI to use your GitLab Self-Managed or GitLab Dedicated account.
1. Select **Authorize**.
1. Complete the authentication process in your terminal, selecting the appropriate options for your needs.

### Personal access token

To authenticate your installation of `glab` with a personal access token:

1. Get a GitLab personal access token with at least the `api`
   and `write_repository` scopes. Use the method appropriate for your instance:
   - For GitLab.com, create one at the [personal access tokens](https://gitlab.com/-/user_settings/personal_access_tokens?scopes=api%2Cwrite_repository) page.
   - For GitLab Self-Managed and GitLab Dedicated, visit `https://gitlab.example.com/-/user_settings/personal_access_tokens?scopes=api,write_repository`,
     modifying `gitlab.example.com` to match the domain name of your instance.
1. Start interactive setup: `glab auth login`
1. Authenticate with the method appropriate for your GitLab instance:
   - For GitLab SaaS, authenticate against `gitlab.com` by reading the token
     from a file: `glab auth login --stdin < myaccesstoken.txt`
   - For GitLab Self-Managed and GitLab Dedicated, authenticate by reading from a file:
     `glab auth login --hostname gitlab.example.com --stdin < myaccesstoken.txt`. This will allow you to perform
     authenticated `glab` commands against your instance when you are in a Git repository with a remote
     matching your instance's host. Alternatively, set `GITLAB_HOST` to direct your command to your instance.
   - Authenticate with token and hostname: `glab auth login --hostname gitlab.example.org --token xxxxx`
     Not recommended for shared environments.
   - Credentials are stored in the global [configuration file](#configuration).

### CI Job Token

To authenticate your installation of `glab` with a CI job token, the `glab` command must be run in a GitLab CI job.
The token is automatically provided by the GitLab Runner via the `CI_JOB_TOKEN` environment variable.

Example:

```shell
glab auth login --job-token $CI_JOB_TOKEN --hostname $CI_SERVER_HOST --api-protocol $CI_SERVER_PROTOCOL
GITLAB_HOST=$CI_SERVER_URL glab release list -R $CI_PROJECT_PATH
```

Endpoints allowing the use of the CI job token are listed in the
[GitLab documentation](https://docs.gitlab.com/ci/jobs/ci_job_token/#job-token-access).

## Configuration

By default, `glab` follows the
[XDG Base Directory Spec](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html).
Configure it globally, locally, or per host:

- **Globally**: run `glab config set --global editor vim`.
  - The global configuration file is available at `~/.config/glab-cli/config.yml`.
  - To override this location, set the `GLAB_CONFIG_DIR` environment variable.
- **The current repository**: run `glab config set editor vim` in any folder in a Git repository.
  - The local configuration file is available at `.git/glab-cli/config.yml` in the current working Git directory.
- **Per host**: run `glab config set editor vim --host gitlab.example.org`, changing
  the `--host` parameter to meet your needs.
  - Per-host configuration info is always stored in the global configuration file, with or without the `global` flag.

### Configure `glab` to use your GitLab Self-Managed or GitLab Dedicated instance

When outside a Git repository, `glab` uses `gitlab.com` by default. For `glab` to default
to your GitLab Self-Managed or GitLab Dedicated instance when you are not in a Git repository, change the host
configuration settings. Use this command, changing `gitlab.example.com` to the domain name
of your instance:

```shell
glab config set -g host gitlab.example.com
```

Setting this configuration enables you to perform commands outside a Git repository while
using your GitLab Self-Managed or GitLab Dedicated instance. For example:

- `glab repo clone group/project`
- `glab issue list -R group/project`

If you don't set a default domain name, you can declare one for the current command with
the `GITLAB_HOST` environment variable, like this:

- `GITLAB_HOST=gitlab.example.com glab repo clone group/project`
- `GITLAB_HOST=gitlab.example.com glab issue list -R group/project`

When inside a Git repository `glab` will use that repository's GitLab host by default. For example `glab issue list`
will list all issues of the current directory's Git repository.

### Configure `glab` to use mTLS certificates

To use a mutual TLS (Mutual Transport Layer Security) certificate with `glab`, edit your global
configuration file (`~/.config/glab-cli/config.yml`) to provide connection information:

```yaml
hosts:
    git.your-domain.com:
        api_protocol: https
        api_host: git.your-domain.com
        token: xxxxxxxxxxxxxxxxxxxxxxxxx
        client_cert: /path/to/client.crt
        client_key: /path/to/client.key
        ca_cert: /path/to/ca-chain.pem
```

- `ca_cert` is optional for mTLS support if you use a publicly signed server certificate.
- `token` is not required if you use a different authentication method.

### Configure `glab` to use self-signed certificates

To configure the GitLab CLI to support GitLab Self-Managed and GitLab Dedicated instances with
self-signed certificates, either:

- Disable TLS verification with:

  ```shell
  glab config set skip_tls_verify true --host gitlab.example.com
  ```

- Add the path to the self signed CA:

  ```shell
  glab config set ca_cert /path/to/server.pem --host gitlab.example.com
  ```

## Environment variables

### GitLab access variables

| Token name         | In `config.yml`                  | Default value if [not set](#configuration) | Description |
|--------------------|----------------------------------|--------------------------------------------|-------------|
| `GITLAB_API_HOST`  | `hosts.<hostname>.api_host`, or `hosts.<hostname>` if empty | Hostname found in the Git URL              | Specify the host where the API endpoint is found. Useful when there are separate (sub)domains or hosts for Git and the API endpoint. |
| `GITLAB_CLIENT_ID` | `hosts.<hostname>.client_id`                             | Client-ID for GitLab.com.                  | A custom Client-ID generated by the GitLab OAuth 2.0 application. |
| `GITLAB_GROUP`     | -                              | -                                        | Default GitLab group used for listing merge requests, issues and variables. Only used if no `--group` option is given. |
| `GITLAB_HOST`      | `host` (this is the default host `glab` will use when the current directory is not a `git` directory)                          | `https://gitlab.com`                       | Alias of `GITLAB_URI`. |
| `GITLAB_REPO`      | -                              | -                                        | Default GitLab repository used for commands accepting the `--repo` option. Only used if no `--repo` option is given. |
| `GITLAB_TOKEN`     | `hosts.<hostname>.token`                          | -                                        | an authentication token for API requests. Setting this avoids being prompted to authenticate and overrides any previously stored credentials. Can be set in the config with `glab config set token xxxxxx`. |
| `GITLAB_URI`       | not applicable                       | not applicable                      | Alias of `GITLAB_HOST`. |

### `glab` configuration variables

| Token name         | In `config.yml` | Default value if [not set](#configuration) | Description |
|--------------------|-----------------|--------------------------------------------|-------------|
| `GLAB_CONFIG_DIR`  | -            | `~/.config/glab-cli/`                      | Directory where the `glab` global configuration file is located. Can be set in the config with `glab config set remote_alias origin`. |
| `BROWSER`          | `browser`       | system default                                        | The web browser to use for opening links. Can be set in the configuration with `glab config set browser mybrowser`. |
| `FORCE_HYPERLINKS` | `display_hyperlinks`             | `false`                                        | Set to `1` to force hyperlinks to be output, even when not outputting to a TTY. |
| `GLAB_SEND_TELEMETRY` | `telemetry`             | `true`                                        | Set to `0` to prevent command usage data from being sent to your GitLab instance. |
| `GLAMOUR_STYLE`    | `glamour_style` | `dark`                                       | Environment variable to set your desired Markdown renderer style. Available options are (`dark`, `light`, `notty`) or set a [custom style](https://github.com/charmbracelet/glamour#styles). |
| `NO_COLOR`         | -            | `true`                                        | Set to any value to avoid printing ANSI escape sequences for color output. |
| `VISUAL`, `EDITOR` | `editor`        | `nano`                                        | (in order of precedence) The editor tool to use for authoring text. Can be set in the config with `glab config set editor vim`. |
| `GLAB_CHECK_UPDATE` | -            | -            | Set to `1`, `TRUE`, or `yes` to force an update check. |

### Other variables

| Token name           | In `config.yml` | Default value if [not set](#configuration) | Description |
|----------------------|-----------------|--------------------------------------------|-------------|
| `DEBUG`              | `debug`            | `false`                                        | Set to `1` or `true` to output more information for each command, like Git commands, expanded aliases, and DNS error details. |
| `GIT_REMOTE_URL_VAR` | not applicable         | not applicable                          | Alias of `REMOTE_ALIAS`. |
| `REMOTE_ALIAS`       | `remote_alias`             | -                                        | `git remote` variable or alias that contains the GitLab URL. Alias: `GIT_REMOTE_URL_VAR` |

### Token and environment variable precedence

GLab uses tokens in this order:

1. Environment variable (`GITLAB_TOKEN`).
1. Configuration file (`$HOME/.config/glab-cli/config.yml`).

### Debugging

When the `DEBUG` environment variable is set to `1` or `true`, `glab` outputs more logging information, including:

- Underlying Git commands.
- Expanded aliases.
- DNS error details.

## Issues

If you have an issue: report it on the [issue tracker](https://gitlab.com/gitlab-org/cli/-/issues)

## Contributing

Feel like contributing? That's awesome! We have a [contributing guide](https://gitlab.com/gitlab-org/cli/-/blob/main/CONTRIBUTING.md) and [Code of conduct](https://gitlab.com/gitlab-org/cli/-/blob/main/CODE_OF_CONDUCT.md) to help guide you.

### Versioning

This project follows the [SemVer](https://github.com/semver/semver) specification.

### Classify version changes

- If deleting a command, changing how it behaves, or adding a new **required** flag, the release must use a new `MAJOR` revision.
- If adding a new command or **optional** flag, the release must use a new `MINOR` revision.
- If fixing a bug, the release must use a new `PATCH` revision.

### Compatibility

We do our best to introduce breaking changes only when releasing a new `MAJOR` version.
Unfortunately, there are situations where this is not possible, and we may introduce
a breaking change in a `MINOR` or `PATCH` version. Some of situations where we may do so:

- If a security issue is discovered, and the solution requires a breaking change,
  we may introduce such a change to resolve the issue and protect our users.
- If a feature was not working as intended, and the bug fix requires a breaking change,
  the bug fix may be introduced to ensure the functionality works as intended.
- When feature behavior is overwhelmingly confusing due to a vague specification
  on how it should work. In such cases, we may refine the specification
  to remove the ambiguity, and introduce a breaking change that aligns with the
  refined specification. For an example of this, see
  [merge request 1382](https://gitlab.com/gitlab-org/cli/-/merge_requests/1382#note_1686888887).
- Experimental features are not guaranteed to be stable, and can be modified or
  removed without a breaking change.

Breaking changes are a last resort, and we try our best to only introduce them when absolutely necessary.

## Inspiration

The GitLab CLI was adopted from [Clement Sam](https://gitlab.com/profclems) in 2022 to serve as the official CLI of GitLab. Over the years the project has been inspired by both the [GitHub CLI](https://github.com/cli/cli) and [Zaq? Wiedmann's](https://gitlab.com/zaquestion) [lab](https://github.com/zaquestion/lab).

Lab has served as the foundation for many of the GitLab CI/CD commands including `ci view` and `ci trace`.
