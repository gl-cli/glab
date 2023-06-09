---
stage: Create
group: Code Review
info: To determine the technical writer assigned to the Stage/Group associated with this page, see https://about.gitlab.com/handbook/product/ux/technical-writing/#assignments
---

<!--
This documentation is auto generated by a script.
Please do not edit this file directly. Run `make gen-docs` instead.
-->

# `glab issue create`

Create an issue

```plaintext
glab issue create [flags]
```

## Aliases

```plaintext
new
```

## Examples

```plaintext
glab issue create
glab issue new
glab issue create -m release-2.0.0 -t "we need this feature" --label important
glab issue new -t "Fix CVE-YYYY-XXXX" -l security --linked-mr 123
glab issue create -m release-1.0.1 -t "security fix" --label security --web --recover

```

## Options

```plaintext
  -a, --assignee usernames     Assign issue to people by their usernames
  -c, --confidential           Set an issue to be confidential. (default false)
  -d, --description string     Supply a description for issue
  -l, --label strings          Add label by name. Multiple labels should be comma separated
      --link-type string       Type for the issue link (default "relates_to")
      --linked-issues ints     The IIDs of issues that this issue links to
      --linked-mr int          The IID of a merge request in which to resolve all issues
  -m, --milestone string       The global ID or title of a milestone to assign
      --no-editor              Don't open editor to enter description. If set to true, uses prompt. (default false)
      --recover                Save the options to a file if the issue fails to be created. If the file exists, the options will be loaded from the recovery file (EXPERIMENTAL)
  -e, --time-estimate string   Set time estimate for the issue
  -s, --time-spent string      Set time spent for the issue
  -t, --title string           Supply a title for issue
      --web                    continue issue creation with web interface
  -w, --weight int             The weight of the issue. Valid values are greater than or equal to 0.
  -y, --yes                    Don't prompt for confirmation to submit the issue
```

## Options inherited from parent commands

```plaintext
      --help              Show help for command
  -R, --repo OWNER/REPO   Select another repository using the OWNER/REPO or `GROUP/NAMESPACE/REPO` format or full URL or git URL
```
