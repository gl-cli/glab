---
stage: Create
group: Code Review
info: To determine the technical writer assigned to the Stage/Group associated with this page, see https://about.gitlab.com/handbook/product/ux/technical-writing/#assignments
---

<!--
This documentation is auto generated by a script.
Please do not edit this file directly. Run `make gen-docs` instead.
-->

# `glab mr merge`

Merge/Accept merge requests

```plaintext
glab mr merge {<id> | <branch>} [flags]
```

## Aliases

```plaintext
accept
```

## Examples

```plaintext
glab mr merge 235
glab mr accept 235
glab mr merge    # Finds open merge request from current branch

```

## Options

```plaintext
      --auto-merge              Set auto-merge (default true)
  -m, --message string          Custom merge commit message
  -r, --rebase                  Rebase the commits onto the base branch
  -d, --remove-source-branch    Remove source branch on merge
      --sha string              Merge Commit sha
  -s, --squash                  Squash commits on merge
      --squash-message string   Custom Squash commit message
  -y, --yes                     Skip submission confirmation prompt
```

## Options inherited from parent commands

```plaintext
      --help              Show help for command
  -R, --repo OWNER/REPO   Select another repository using the OWNER/REPO or `GROUP/NAMESPACE/REPO` format or full URL or git URL
```
