---
stage: Create
group: Code Review
info: To determine the technical writer assigned to the Stage/Group associated with this page, see https://about.gitlab.com/handbook/product/ux/technical-writing/#assignments
---

<!--
This documentation is auto generated by a script.
Please do not edit this file directly, check cmd/gen-docs/docs.go.
-->

# `glab config`

Set and get glab settings

## Synopsis

Get and set key/value strings.

Current respected settings:

- token: Your GitLab access token, defaults to environment variables
- gitlab_uri: if unset, defaults to `https://gitlab.com`
- browser: if unset, defaults to environment variables
- editor: if unset, defaults to environment variables.
- visual: alternative for editor. if unset, defaults to environment variables.
- glamour_style: Your desired Markdown renderer style. Options are dark, light, notty. Custom styles are allowed using [glamour](https://github.com/charmbracelet/glamour#styles)
- glab_pager: Your desired pager command to use (e.g. less -R)

## Options

```plaintext
  -g, --global   use global config file
```

## Options inherited from parent commands

```plaintext
      --help   Show help for command
```

## Subcommands

- [get](get.md)
- [init](init.md)
- [set](set.md)