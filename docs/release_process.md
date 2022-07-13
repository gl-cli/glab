# Release process

You need to perform the following steps to release a new version of the cli.

1. Do a quick test of the cli in your local development. At this stage, you are only verifying there is no complete failure of the cli.
1. Tag the latest commit on `main` (e.g. `git tag v1.22.1`)
1. `git push origin main` and `git push --tags`
1. Manually add attribution to the changelog - you have to edit the release entry on the releases page https://gitlab.com/gitlab-org/cli/-/releases.

## Access to distribution channels

TODO: Here we'll add information about how we set up release for Homberew, WinGet and others: https://gitlab.com/groups/gitlab-org/-/epics/8251

## Setting up CI for releasing

For automated testing, you need to set up credentials for unit testing: https://gitlab.com/gitlab-org/cli/-/issues/1020

For releasing, you also need to add a `GITLAB_TOKEN_RELEASE`. This is how you create this token:

1. Go to Settings -> Access Tokens (https://gitlab.com/gitlab-org/cli/-/settings/access_tokens)
1. Generate a new project token with `api` scope and `Maintainer` role.
1. Add the new token as `GITLAB_TOKEN_RELEASE` **protected** and **masked** CI variables.
