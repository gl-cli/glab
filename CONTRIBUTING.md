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

* Check existing issues to verify that the bug or feature request has not already been submitted.
* Open an issue if things aren't working as expected.
* Open an issue to propose a significant change.
* Open an issue to propose a feature.
* Open a merge request to fix a bug.
* Open a merge request to fix documentation about a command.
* Open a merge request for an issue and leave a comment claiming it.

Please avoid:

* Opening merge requests for issues marked `blocked`.
* Opening merge requests for documentation for a new command specifically. Manual pages are auto-generated from source after every release
## Code of Conduct

We want to create a welcoming environment for everyone who is interested in contributing. Visit our [Code of Conduct page](https://about.gitlab.com/community/contribute/code-of-conduct/) to learn more about our commitment to an open and welcoming environment.

## Getting Started
### Building the project

Prerequisites:
- Go 1.16+

Build with: `make` or `go build -o bin/glab ./cmd/glab/main.go`

Run the new binary as: `./bin/glab`

Run tests with: `make test` or `go test ./...`

> WARNING: Do not run `make test` outside of an isolated environment, it will overwrite your global config.

### Submitting a merge request

1. Create a new branch: `git checkout -b my-branch-name`
1. Make your change, add tests, and ensure tests pass
1. Submit a merge request

## Commit Messages

### TL;DR: Your commit message should be semantic

A commit message consists of a header, a body and a footer, separated by a blank line.

Any line of the commit message cannot be longer than 100 characters! This allows the message to be easier to read as well as in various git tools.

```sh
<type>[optional scope]: <description>
<BLANK LINE>
[optional body]
<BLANK LINE>
<footer>
```

### Message Header
Ideally, the commit message heading which contains the description, should not be more than 50 characters

The message header is a single line that contains a succinct description of the change containing a type, an optional scope, and a subject.

#### `<type>`

This describes the kind of change that this commit is providing

- feat (feature)
- fix (bug fix)
- docs (documentation)
- style (formatting, missing semicolons, …)
- refactor(restructuring codebase)
- test (when adding missing tests)
- chore (maintain)

#### `<scope>`

Scope can be anything specifying the place of the commit change. For example events, kafka, userModel, authorization, authentication, loginPage, etc

#### `<description>`

This is a very short description of the change

* `use imperative, present tense: “change” not “changed” nor “changes”`
* `don't capitalize the first letter`
* `no dot (.) at the end`

### Message Body

- just as in subject use imperative, present tense: “change” not “changed” nor “changes”
- includes motivation for the change and contrasts with previous behavior

<http://365git.tumblr.com/post/3308646748/writing-git-commit-messages>

<http://tbaggery.com/2008/04/19/a-note-about-git-commit-messages.html>

### Message Footer

Finished, fixed or delivered stories should be listed on a separate line in the footer prefixed with "Finishes", "Fixes" , or "Delivers" keyword like this:

`[(Finishes|Fixes|Delivers) #ISSUE_ID]`

### Message Example

```sh
feat(kafka): implement exactly once delivery

- ensure every event published to kafka is delivered exactly once
- implement error handling for failed delivery

Delivers #065
```

```sh
fix(login): allow provided user preferences to override default preferences

- This allows the preferences associated with a user account to override and customize the default app preference like theme, timezone e.t.c

Fixes #025
```

