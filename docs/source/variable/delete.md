---
stage: Create
group: Code Review
info: To determine the technical writer assigned to the Stage/Group associated with this page, see https://about.gitlab.com/handbook/product/ux/technical-writing/#assignments
---

<!--
This documentation is auto generated by a script.
Please do not edit this file directly, check cmd/gen-docs/docs.go.
-->

# `glab variable delete`

Delete a project or group variable

```plaintext
glab variable delete <key> [flags]
```

## Examples

```plaintext
glab variable delete VAR_NAME
glab variable delete VAR_NAME --scope=prod
glab variable delete VARNAME -g mygroup

```

## Options

```plaintext
  -g, --group string   Delete variable from a group
  -s, --scope string   The environment_scope of the variable. All (*), or specific environments (default "*")
```

## Options inherited from parent commands

```plaintext
      --help              Show help for command
  -R, --repo OWNER/REPO   Select another repository using the OWNER/REPO or `GROUP/NAMESPACE/REPO` format or full URL or git URL
```