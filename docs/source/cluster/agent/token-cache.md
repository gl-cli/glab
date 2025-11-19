# `glab cluster agent token-cache`

Manage cached GitLab Agent tokens created by `glab cluster agent get-token`.

This command group allows you to list and clear tokens that are cached locally
in the keyring and filesystem cache.

## Subcommands

- [`glab cluster agent token-cache list`](list.md) - List cached tokens
- [`glab cluster agent token-cache clear`](clear.md) - Clear cached tokens

## Cache Storage

The GitLab CLI caches agent tokens in two locations:

1. **Keyring** - Uses the system keyring (Windows Credential Manager, macOS Keychain, Linux Secret Service)
2. **Filesystem** - Stores tokens in the user's cache directory as encrypted files

The cache improves performance by avoiding the need to create new tokens for each `kubectl` operation when using `glab cluster agent update-kubeconfig`.

## Cache Key Format

Cached tokens are stored using a key format that includes:

- Base64-encoded GitLab instance URL
- Agent ID

This ensures tokens are properly isolated by GitLab instance and agent.
