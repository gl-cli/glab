---
stage: Create
group: Code Review
info: To determine the technical writer assigned to the Stage/Group associated with this page, see https://about.gitlab.com/handbook/product/ux/technical-writing/#assignments
---

<!--
This documentation is auto generated by a script.
Please do not edit this file directly. Run `make gen-docs` instead.
-->

# `glab label create`

Create labels for a repository or project.

```plaintext
glab label create [flags]
```

## Aliases

```plaintext
new
```

## Examples

```console
$ glab label create
$ glab label new
$ glab label create -R owner/repo

```

## Options

```plaintext
  -c, --color string         Color of the label, in plain or HEX code. (default "#428BCA")
  -d, --description string   Label description.
  -n, --name string          Name of the label.
```

## Options inherited from parent commands

```plaintext
  -h, --help              Show help for this command.
  -R, --repo OWNER/REPO   Select another repository. Can use either OWNER/REPO or `GROUP/NAMESPACE/REPO` format. Also accepts full URL or Git URL.
```
