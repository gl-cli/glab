---
stage: Create
group: Code Review
info: To determine the technical writer assigned to the Stage/Group associated with this page, see https://about.gitlab.com/handbook/product/ux/technical-writing/#assignments
---

<!--
This documentation is auto generated by a script.
Please do not edit this file directly. Run `make gen-docs` instead.
-->

# `glab check-update`

Check for latest glab releases.

## Synopsis

Checks for new versions every 24 hours after any 'glab' command is run. Does not recheck if the most recent recheck is less than 24 hours old.

To override the recheck behavior and force an update check, set the GLAB_CHECK_UPDATE environment variable to 'true'.

To disable the update check entirely, run 'glab config set check_update false'.
To re-enable the update check, run 'glab config set check_update true'.

```plaintext
glab check-update [flags]
```

## Aliases

```plaintext
update
```

## Options inherited from parent commands

```plaintext
  -h, --help   Show help for this command.
```
