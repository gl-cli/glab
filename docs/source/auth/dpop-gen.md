---
stage: Create
group: Code Review
info: To determine the technical writer assigned to the Stage/Group associated with this page, see https://about.gitlab.com/handbook/product/ux/technical-writing/#assignments
---

<!--
This documentation is auto generated by a script.
Please do not edit this file directly. Run `make gen-docs` instead.
-->

# `glab auth dpop-gen`

Generates a DPoP (demonstrating-proof-of-possession) proof JWT. (Experimental.)

## Synopsis

Demonstrating-proof-of-possession (DPoP) is a technique to
cryptographically bind personal access tokens to their owners. This command provides
the tools to manage the client aspects of DPoP. It generates a DPoP proof JWT
(JSON Web Token).

Prerequisites:

- You must have a SSH key pair in RSA, ed25519, or ECDSA format.
- You have enabled DPoP for your account, as described in the [GitLab documentation.](https://docs.gitlab.com/user/profile/personal_access_tokens/#require-dpop-headers-with-personal-access-tokens)

Use the JWT in combination with a Personal Access Token (PAT) to authenticate to
the GitLab API. Your JWT remains valid for 5 minutes. After it expires, you must
generate another token. Your SSH private key is then used to sign the JWT.

This feature is experimental. It might be broken or removed without any prior notice.
Read more about what experimental features mean at
<https://docs.gitlab.com/policy/development_stages_support/>

Use experimental features at your own risk.

```plaintext
glab auth dpop-gen [flags]
```

## Examples

```console
# Generate a DPoP JWT for authentication to GitLab
$ glab auth dpop-gen [flags]
$ glab auth dpop-gen --private-key "~/.ssh/id_rsa" --pat "glpat-xxxxxxxxxxxxxxxxxxxx"

# No PAT required if you previously used the 'glab auth login' command with a PAT
$ glab auth dpop-gen --private-key "~/.ssh/id_rsa"

# Generate a DPoP JWT for a different GitLab instance
$ glab auth dpop-gen --private-key "~/.ssh/id_rsa" --hostname "https://gitlab.com"

```

## Options

```plaintext
      --hostname string      The hostname of the GitLab instance to authenticate with. Defaults to 'gitlab.com'. (default "gitlab.com")
      --pat string           Personal Access Token (PAT) to generate a DPoP proof for. Defaults to the token set with 'glab auth login'. Returns an error if both are empty.
  -p, --private-key string   Location of the private SSH key on the local system.
```

## Options inherited from parent commands

```plaintext
  -h, --help   Show help for this command.
```
