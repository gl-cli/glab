---
stage: Create
group: Code Review
info: To determine the technical writer assigned to the Stage/Group associated with this page, see https://about.gitlab.com/handbook/product/ux/technical-writing/#assignments
---

<!--
This documentation is auto generated by a script.
Please do not edit this file directly. Run `make gen-docs` instead.
-->

# `glab repo search`

Search for GitLab repositories and projects by name.

```plaintext
glab repo search [flags]
```

## Aliases

```plaintext
find
lookup
```

## Examples

```console
$ glab project search -s "title"
$ glab repo search -s "title"
$ glab project find -s "title"
$ glab project lookup -s "title"

```

## Options

```plaintext
  -F, --output string   Format output as: text, json. (default "text")
  -p, --page int        Page number. (default 1)
  -P, --per-page int    Number of items to list per page. (default 20)
  -s, --search string   A string contained in the project name.
```

## Options inherited from parent commands

```plaintext
  -h, --help   Show help for this command.
```
