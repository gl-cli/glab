---
stage: Create
group: Code Review
info: To determine the technical writer assigned to the Stage/Group associated with this page, see https://about.gitlab.com/handbook/product/ux/technical-writing/#assignments
---

<!--
This documentation is auto generated by a script.
Please do not edit this file directly, check cmd/gen-docs/docs.go.
-->

# `glab repo contributors`

Get repository contributors list.

```plaintext
glab repo contributors [flags]
```

## Examples

```plaintext
$ glab repo contributors

$ glab repo contributors -R gitlab-com/www-gitlab-com
#=> Supports repo override

```

## Options

```plaintext
  -o, --order string      Return contributors ordered by name, email, or commits (orders by commit date) fields (default "commits")
  -p, --page int          Page number (default 1)
  -P, --per-page int      Number of items to list per page. (default 30)
  -R, --repo OWNER/REPO   Select another repository using the OWNER/REPO or `GROUP/NAMESPACE/REPO` format or full URL or git URL
  -s, --sort string       Return contributors sorted in asc or desc order
```

## Options inherited from parent commands

```plaintext
      --help   Show help for command
```