---
stage: Create
group: Code Review
info: To determine the technical writer assigned to the Stage/Group associated with this page, see https://about.gitlab.com/handbook/product/ux/technical-writing/#assignments
---

<!--
This documentation is auto generated by a script.
Please do not edit this file directly. Run `make gen-docs` instead.
-->

# `glab issue update`

Update issue

```plaintext
glab issue update <id> [flags]
```

## Examples

```console
$ glab issue update 42 --label ui,ux
$ glab issue update 42 --unlabel working

```

## Options

```plaintext
  -a, --assignee strings     Assign users by username. Prefix with '!' or '-' to remove from existing assignees, or '+' to add new. Otherwise, replace existing assignees with these users.
  -c, --confidential         Make issue confidential
  -d, --description string   Issue description. Set to "-" to open an editor.
      --due-date string      A date in 'YYYY-MM-DD' format.
  -l, --label strings        Add labels.
      --lock-discussion      Lock discussion on issue.
  -m, --milestone string     Title of the milestone to assign Set to "" or 0 to unassign.
  -p, --public               Make issue public.
  -t, --title string         Title of issue.
      --unassign             Unassign all users.
  -u, --unlabel strings      Remove labels.
      --unlock-discussion    Unlock discussion on issue.
  -w, --weight int           Set weight of the issue.
```

## Options inherited from parent commands

```plaintext
  -h, --help              Show help for this command.
  -R, --repo OWNER/REPO   Select another repository. Can use either OWNER/REPO or `GROUP/NAMESPACE/REPO` format. Also accepts full URL or Git URL.
```
