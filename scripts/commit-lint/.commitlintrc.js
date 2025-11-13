// Commitlint configuration for local git hooks
// This is used by lefthook's commit-msg hook
// For CI, see scripts/commit-lint/lint.js which has additional MR-specific logic

export default {
  extends: ['@commitlint/config-conventional'],
  rules: {
    'header-max-length': [2, 'always', 100],
    'body-leading-blank': [2, 'always'],
    'footer-leading-blank': [2, 'always'],
    'subject-case': [0], // Disabled
    'body-max-line-length': [1, 'always', 100],
  },
  ignores: [
    // Same ignores as CI
    (message) => /^[Rr]evert .*/.test(message),
    (message) => /^(?:fixup|squash)!/.test(message),
    (message) => /^Merge branch/.test(message),
    (message) => /^\d+\.\d+\.\d+/.test(message),
  ],
  helpUrl: 'https://gitlab.com/gitlab-org/gitlab-vscode-extension/-/blob/main/docs/developer/commits.md',
};
