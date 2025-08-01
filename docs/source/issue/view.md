---
stage: Create
group: Code Review
info: To determine the technical writer assigned to the Stage/Group associated with this page, see https://about.gitlab.com/handbook/product/ux/technical-writing/#assignments
---

<!--
This documentation is auto generated by a script.
Please do not edit this file directly. Run `make gen-docs` instead.
-->

# `glab issue view`

Display the title, body, and other information about an issue.

```plaintext
glab issue view <id> [flags]
```

## Aliases

```plaintext
show
```

## Examples

```console
$ glab issue view 123
$ glab issue show 123
$ glab issue view --web 123
$ glab issue view --comments 123
$ glab issue view https://gitlab.com/NAMESPACE/REPO/-/issues/123

```

## Options

```plaintext
  -c, --comments        Show issue comments and activities.
  -F, --output string   Format output as: text, json. (default "text")
  -p, --page int        Page number. (default 1)
  -P, --per-page int    Number of items to list per page. (default 20)
  -s, --system-logs     Show system activities and logs.
  -w, --web             Open issue in a browser. Uses the default browser, or the browser specified in the $BROWSER variable.
```

## Options inherited from parent commands

```plaintext
  -h, --help              Show help for this command.
  -R, --repo OWNER/REPO   Select another repository. Can use either OWNER/REPO or `GROUP/NAMESPACE/REPO` format. Also accepts full URL or Git URL.
```
