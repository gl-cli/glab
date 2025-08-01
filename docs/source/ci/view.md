---
stage: Create
group: Code Review
info: To determine the technical writer assigned to the Stage/Group associated with this page, see https://about.gitlab.com/handbook/product/ux/technical-writing/#assignments
---

<!--
This documentation is auto generated by a script.
Please do not edit this file directly. Run `make gen-docs` instead.
-->

# `glab ci view`

View, run, trace, log, and cancel CI/CD job's current pipeline.

## Synopsis

Supports viewing, running, tracing, and canceling jobs.

Use arrow keys to navigate jobs and logs.

- 'Enter' to toggle through a job's logs / traces, or display a child pipeline. Trigger jobs are marked with a '»'.
- 'Esc' or 'q' to close the logs or trace, or return to the parent pipeline.
- 'Ctrl+R', 'Ctrl+P' to run, retry, or play a job. Use 'Tab' or arrow keys to navigate the modal, and 'Enter' to confirm.
- 'Ctrl+D' to cancel a job. If the selected job isn't running or pending, quits the CI/CD view.
- 'Ctrl+Q' to quit the CI/CD view.
- 'Ctrl+Space' to suspend application and view the logs. Similar to 'glab pipeline ci trace'.
Supports vi style bindings and arrow keys for navigating jobs and logs.

```plaintext
glab ci view [branch/tag] [flags]
```

## Examples

```console
# Uses current branch
$ glab pipeline ci view

# Get latest pipeline on master branch
$ glab pipeline ci view master

# just like the second example
$ glab pipeline ci view -b master

# Get latest pipeline on master branch of profclems/glab repo
$ glab pipeline ci view -b master -R profclems/glab

```

## Options

```plaintext
  -b, --branch string   Check pipeline status for a branch or tag. Defaults to the current branch.
  -w, --web             Open pipeline in a browser. Uses default browser, or browser specified in BROWSER variable.
```

## Options inherited from parent commands

```plaintext
  -h, --help              Show help for this command.
  -R, --repo OWNER/REPO   Select another repository. Can use either OWNER/REPO or `GROUP/NAMESPACE/REPO` format. Also accepts full URL or Git URL.
```
