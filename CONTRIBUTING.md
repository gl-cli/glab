# Contributing Guide

## Developer Certificate of Origin + License

Contributions to this repository are subject to the [Developer Certificate of Origin](https://docs.gitlab.com/ee/legal/developer_certificate_of_origin.html#developer-certificate-of-origin-version-11).

All Documentation content that resides under the [docs/ directory](/docs) of this
repository is licensed under Creative Commons:
[CC BY-SA 4.0](https://creativecommons.org/licenses/by-sa/4.0/).

_This notice should stay as the first item in the CONTRIBUTING.md file._

---

Thank you for your interest in contributing to the GitLab CLI! This guide details how to contribute
to this extension in a way that is easy for everyone. These are mostly guidelines, not rules.
Use your best judgement, and feel free to propose changes to this document in a merge request.

Please do:

- Check existing issues to verify that the bug or feature request has not already been submitted.
- Open an issue if things aren't working as expected.
- Open an issue to propose a significant change.
- Open an issue to propose a feature.
- Open a merge request to fix a bug.
- Open a merge request to fix documentation about a command.
- Open a merge request for an issue and leave a comment claiming it.

Please avoid:

- Opening merge requests for issues marked `blocked`.
- Opening merge requests for documentation for a new command specifically. Manual pages are auto-generated from source after every release

## Code of Conduct

We want to create a welcoming environment for everyone who is interested in contributing. Visit our [Code of Conduct page](https://about.gitlab.com/community/contribute/code-of-conduct/) to learn more about our commitment to an open and welcoming environment.

## Maintainership

If you are a GitLab team member that is interested in becoming a maintainer of the CLI, we follow these basic steps [described in the handbook](https://about.gitlab.com/handbook/engineering/workflow/code-review/#accelerated-maintainer-onboarding-for-smaller-projects):

- Familiarize yourself with the codebase and past reviews.
- When you're ready, add yourself as a reviewer in your [team page](https://gitlab.com/gitlab-com/www-gitlab-com/-/blob/master/data/team_members/person/).
- After the current maintainers feel confident you're ready to be a maintainer, you're added
  to the project, and can update your team page again.

## Getting Started

### Building the project

Prerequisites:

- Go 1.18+

Build with: `make` or `go build -o bin/glab ./cmd/glab/main.go`

Run the new binary as: `./bin/glab`

### Running tests

Run tests with: `go test ./...` or `make test`.

There are some integration tests that perform API requests to a real GitLab instance.
If environment variables `GITLAB_TEST_HOST` and `GITLAB_TOKEN` are not set, the integration tests will fail in CI or will be skipped locally.
Integration tests use the `_Integration` test suffix and use the `_integration_test.go` file suffix.

`GITLAB_TOKEN` must be a
[personal access token](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html).

### Submitting a merge request

1. Create a new branch: `git checkout -b my-branch-name`
1. Make your change, add tests, and ensure tests pass
1. Submit a merge request

### Formatting your code

We use [`golangci-lint`](https://golangci-lint.run/) to lint and format
the code in this project. The linter configuration can be seen
[here](https://gitlab.com/gitlab-org/cli/-/blob/main/.golangci.yml).

Additional details about code style and format are in the
[go guide](https://docs.gitlab.com/ee/development/go_guide/#code-style-and-format).

## Commit Messages

Each commit message consists of a **header**, a **body**, and a **footer**. The header has a special format that includes a **type**, a **scope**, and a **description**:

```plaintext
<type>(<scope>): <description>
<BLANK LINE>
<body>
<BLANK LINE>
<footer>
```

Each line in the commit message should be no longer than 72 characters.

### Message Header

The message header is mandatory, and should be a single line that contains a succinct description of the change containing a type, an optional scope, and a description. Ideally, it should not be more than 50 characters in length.

Following these conventions results in a clear changelog for every version.

It's generally a good idea to follow the conventions for your MR's title as well as for commit messages. This way, if your merge request is squashed upon merge, the maintainer can use its title as the final commit message, creating a properly-formatted history.

If your MR contains multiple commits but only one logical change, the [Squash commits when merge request is accepted](https://gitlab.com/help/user/project/merge_requests/squash_and_merge) option (enabled by default in this project) will allow GitLab to use the MR title as the commit message.

#### `<type>`

This describes the kind of change that this commit is providing

- **feat:** A new feature (adding a new component, providing new variants for an existing component, etc.).
- **fix:** A bug fix (correcting a styling issue, addressing a bug in a component's API, etc.).
  When updating non-dev dependencies, mark your changes with the `fix:` type.
- **docs:** Documentation-only changes.
- **style:** Changes that do not affect the meaning of the code
  (whitespace, formatting, missing semicolons, etc). _Not_ to be used for UI changes as those are
  meaningful changes, consider using `feat:` of `fix:` instead.
- **refactor:** A code change that neither fixes a bug nor adds a feature.
- **perf:** A code change that improves performance.
- **test:** Adding missing tests or correcting existing tests.
- **build:** Changes that affect the build system.
- **ci:** Changes to our CI/CD configuration files and scripts.
- **chore:** Other changes that don't modify source or test files. Use this type when adding or
  updating dev dependencies.
- **revert:** Reverts a previous commit.

Each commit type can have an optional scope to specify the place of the commit change: `type(scope):`. It is up to you to add or omit a commit's scope. When a commit affects a specific component, use the component's PascalCase name as the commit's scope. For example:

```plaintext
feat(statusbar): automatically switch pipelines
```

#### `<scope>`

Scope can be anything specifying the place of the commit change. For example events, kafka, userModel, authorization, authentication, loginPage, etc

#### `<description>`

This is a very short description of the change

- `use imperative, present tense: “change” not “changed” nor “changes”`
- `don't capitalize the first letter`
- `no dot (.) at the end`

### Message Body

Just as in the description, use imperative, present tense: “change” not “changed” nor “changes.” Include motivation for the change and contrast it with previous behavior.

#### More info on writing good git commit messages

- [Writing Git commit messages](http://365git.tumblr.com/post/3308646748/writing-git-commit-messages)
- [A Note About Git Commit Messages](http://tbaggery.com/2008/04/19/a-note-about-git-commit-messages.html)

### Message Footer

Finished, fixed or delivered stories should be listed on a separate line in the footer prefixed with "Finishes", "Fixes" , or "Delivers" keyword like this:

`[(Finishes|Fixes|Delivers) #ISSUE_ID]`

### Message Example

```shell
feat(kafka): implement exactly once delivery

- ensure every event published to kafka is delivered exactly once
- implement error handling for failed delivery

Delivers #065
```

```shell
fix(login): allow provided user preferences to override default preferences

- This allows the preferences associated with a user account to
override and customize the default app preferences like theme or timezone.

Fixes #025
```

### Linting

We use the following logic to lint your MR's commit messages:

```mermaid
graph TD
A{Are there multiple commits?} --no--> B[Commit must be valid]
A --yes--> C
C{Is MR set to be squashed?} --no--> D[Every commit must be valid]
C --yes--> E[MR title must be valid]
```
