---
stage: Create
group: Code Review
info: To determine the technical writer assigned to the Stage/Group associated with this page, see https://about.gitlab.com/handbook/product/ux/technical-writing/#assignments
---

<!--
This documentation is auto generated by a script.
Please do not edit this file directly. Run `make gen-docs` instead.
-->

# `glab cluster agent get-token`

Create and return a k8s_proxy-scoped personal access token to authenticate with a GitLab Agents for Kubernetes.

## Synopsis

Create and return a k8s_proxy-scoped personal access token to authenticate with a GitLab Agents for Kubernetes.

This command creates a personal access token that is valid until the end of the current day.
You might receive an email from your GitLab instance that a new personal access token has been created.

```plaintext
glab cluster agent get-token [flags]
```

## Options

```plaintext
  -a, --agent int                        The numerical Agent ID to connect to.
  -c, --cache-mode string                Mode to use for caching the token (allowed: keyring-filesystem-fallback, force-keyring, force-filesystem, no) (default "force-filesystem")
      --token-expiry-duration duration   Duration for how long the generated tokens should be valid for. Minimum is 1 day and the effective expiry is always at the end of the day, the time is ignored. (default 24h0m0s)
```

## Options inherited from parent commands

```plaintext
  -h, --help              Show help for this command.
  -R, --repo OWNER/REPO   Select another repository. Can use either OWNER/REPO or `GROUP/NAMESPACE/REPO` format. Also accepts full URL or Git URL.
```
