---
stage: Create
group: Code Review
info: To determine the technical writer assigned to the Stage/Group associated with this page, see https://about.gitlab.com/handbook/product/ux/technical-writing/#assignments
---

<!--
This documentation is auto generated by a script.
Please do not edit this file directly. Run `make gen-docs` instead.
-->

# `glab job artifact`

Download all artifacts from the last pipeline.

```plaintext
glab job artifact <refName> <jobName> [flags]
```

## Aliases

```plaintext
push
```

## Examples

```console
$ glab job artifact main build
$ glab job artifact main deploy --path="artifacts/"
$ glab job artifact main deploy --list-paths

```

## Options

```plaintext
  -l, --list-paths    Print the paths of downloaded artifacts.
  -p, --path string   Path to download the artifact files. (default "./")
```

## Options inherited from parent commands

```plaintext
  -h, --help              Show help for this command.
  -R, --repo OWNER/REPO   Select another repository. Can use either OWNER/REPO or `GROUP/NAMESPACE/REPO` format. Also accepts full URL or Git URL.
```
