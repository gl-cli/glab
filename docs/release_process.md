# Release process

To release a new version of the CLI, you must:

1. Do a quick test of the CLI in your local development. At this stage, you are only verifying there is no complete failure of the CLI.
1. Tag the latest commit on `main` (such as `git tag v1.22.1`).
1. Push with these commands: `git push origin main` and `git push origin v1.22.1`
1. Manually add attribution to the changelog by editing the release entry on the [releases page](https://gitlab.com/gitlab-org/cli/-/releases).

## Notifying maintainers

If the release is time-sensitive, such as for a security release, consider emailing
the maintainers of the community-maintained channels. Contacts for the different maintainers
are available in CLI 1Password Vault in the **Maintainers of Community packages** note.

## Access to distribution channels

### Homebrew

Homebrew releases were [automated by the CI build](https://gitlab.com/gitlab-org/cli/-/merge_requests/1137) in 15.9.
These manual instructions are provided if the automation fails.

Prerequisites:

- An account on GitHub. (Any account is fine.)

To manually update the version available through Homebrew:

1. Generate a new token with the `repo`, `workflow`, and `gist` scopes  [using this link](https://github.com/settings/tokens/new?scopes=gist,repo,workflow&description=Homebrew). If you have an existing token with the correct scope, you can use it instead.
1. On the [**Releases** page for this project](https://gitlab.com/gitlab-org/cli/-/releases), identify the release version you want to publish.
1. In the **Assets** area for the release, download the packaged source code (`Source code (tar.gz)`) for this release.
1. To compute the SHA256 checksum, run `sha256sum cli-v1.23.0.tar.gz`.
1. Create a pull request with the update with this command, modifying the source code URL and SHA as needed:

   ```shell
   brew bump-formula-pr --strict glab \
   --url="https://gitlab.com/gitlab-org/cli/-/archive/v1.23.0/cli-v1.23.0.tar.gz" \
   --sha256=4fe9dcecc5e601a849454a3608b47d709b11e2df81527f666b169e0dd362d7df
   ```

1. When the pull request is merged, the update is complete.

### Snapcraft

The `latest/edge` channel for Snapcraft can be automatically built from a Git repository,
but as of 2024-07-22 it must be hosted on GitHub. We've configured the [GitHub fork](https://github.com/gl-cli/glab)
as a push mirror, and receives changes to `main` only. To see the configuration:

1. On the left sidebar, select **Settings > Repository**.
1. Expand **Mirroring Repositories** and find the GitHub mirror.

Most `snap` users use the `latest/stable` release channel. To release to it:

1. Sign in to `snapcraft.io`. The credentials are available in 1Password.
1. In the [releases page of the listing](https://snapcraft.io/glab/releases),
   promote one of the `latest/edge` builds.

1Password contains credentials for both `snapcraft.io` and the GitHub fork.

### Scoop

No manual action required.

The `glab` project was moved to GitLab in [pull request 4168](https://github.com/ScoopInstaller/Main/pull/4168/files). Scoop uses the [autoupdate URL](https://github.com/ScoopInstaller/Main/pull/4168/files#diff-f454f19e58d4c978be55818fa3c6ad5e1424e81fbb0b693dca0b76cc879f5457L21) for updating to newer versions.

### WinGet

The SHA is not computed during build, and must be computed manually as part of your update.
For more information, see [Release CI: compute SHA256 of the Windows installer](https://gitlab.com/gitlab-org/cli/-/issues/1133)).

Prerequisites:

- You must have a GitHub account.
- You must have signed Microsoft's Contributor License Agreement (CLA).
- You have read the GitLab policy for [Contributing to a third-party project on behalf of GitLab](https://handbook.gitlab.com/handbook/engineering/open-source/). (The confidential, internal [legal issue](https://gitlab.com/gitlab-com/legal-and-compliance/-/issues/1286) is also available.)

To update the WinGet package:

1. On the [**Releases** page for this project](https://gitlab.com/gitlab-org/cli/-/releases), identify the release version you want to publish.
1. In the **Assets** area for the release, identify the Windows installer package (the filename should end in `_installer.exe`) and download it.
1. Compute the SHA256 checksum for the file by running `sha256sum filename.exe`, replacing `filename` with the name of release installer you downloaded in the previous step. For example:

   ```shell
   $ sha256sum glab_1.23.1_Windows_x86_64_installer.exe

   36f9d45f68ea53cbafdbe96ba4206e4412abb4c2b8f00ba667054523bd7cc89e  glab_1.23.1_Windows_x86_64_installer.exe
   ```

1. Copy the SHA from the result.
1. On GitHub, create a pull request in the `microsoft/winget-pkgs` project. Use the
   [pull request to update to version 1.23.1](https://github.com/microsoft/winget-pkgs/pull/90349) as an example.

## Setting up CI/CD for releasing

For automated testing, you need to [set up credentials](https://gitlab.com/groups/gitlab-org/-/epics/8251) for unit testing.

For releasing, you also need to add a `GITLAB_TOKEN_RELEASE`. This is how you create this token:

1. Go to Settings -> [Access Tokens](https://gitlab.com/gitlab-org/cli/-/settings/access_tokens)
1. Generate a new project token with `api` scope and `Maintainer` role.
1. Add the new token as `GITLAB_TOKEN_RELEASE` **protected** and **masked** CI/CD variables.
