---
stage: Create
group: Code Review
info: To determine the technical writer assigned to the Stage/Group associated with this page, see https://about.gitlab.com/handbook/product/ux/technical-writing/#assignments
---

<!--
This documentation is auto generated by a script.
Please do not edit this file directly. Run `make gen-docs` instead.
-->

# `glab ask git`

Generate Git commands from natural language (Experimental).

## Synopsis

Generate Git commands from natural language.

This experimental feature converts natural language descriptions into
executable Git commands.

We'd love your feedback in [issue 409636](https://gitlab.com/gitlab-org/gitlab/-/issues/409636).

```plaintext
glab ask git <prompt> [flags]
```

## Examples

```plaintext
$ glab ask git list last 10 commit titles
# => A list of Git commands to show the titles of the latest 10 commits with an explanation and an option to execute the commands.

```

## Options inherited from parent commands

```plaintext
      --help   Show help for command
```
