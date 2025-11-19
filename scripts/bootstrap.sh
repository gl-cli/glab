#!/usr/bin/env bash
# Bootstrap script to install development tools for git hooks
# This script is called by `make bootstrap`

set -e  # Exit on error

echo "Installing development tools..."
echo ""

# Check if mise is installed
if ! command -v mise > /dev/null 2>&1; then
    echo "⚠️  mise not found. Install mise from https://mise.jdx.dev/"
    echo "   Alternatively, manually install tools listed in .tool-versions"
    exit 1
fi

# Install tools via mise
echo "Installing tools via mise..."
if ! mise install; then
    echo "❌ mise install failed"
    exit 1
fi

echo ""

# Install commitlint dependencies
echo "Installing commitlint dependencies..."
if ! command -v npm > /dev/null 2>&1; then
    echo "⚠️  npm not found (should be installed by mise if node is in .tool-versions)"
    exit 1
fi

if ! (cd scripts/commit-lint && npm install); then
    echo "❌ npm install failed"
    exit 1
fi

echo ""
echo "✓ Bootstrap complete!"
echo ""
echo "To enable git hooks, run: lefthook install"
