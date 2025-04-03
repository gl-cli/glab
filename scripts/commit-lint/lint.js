import read from '@commitlint/read';
import lint from '@commitlint/lint';
import format from '@commitlint/format';
import config from '@commitlint/config-conventional';

// You can test the script by setting these environment variables
const {
  CI_MERGE_REQUEST_DIFF_BASE_SHA, // refers to the main branch
  CI_MERGE_REQUEST_SQUASH_ON_MERGE, // true if the squash MR checkbox is ticked
  CI_MERGE_REQUEST_TITLE, // MR Title
  CI_MERGE_REQUEST_EVENT_TYPE, // equal to 'merge_train' if the pipeline is a merge train pipeline
  CI, // true when script is run in a CI/CD pipeline
  LAST_MR_COMMIT, // This variable is created by `lint.sh` script. It represents the MR commit that's direct parent of the newly created merge commit.
} = process.env;

const urlSemanticRelease =
  'https://gitlab.com/gitlab-org/gitlab-vscode-extension/-/blob/main/docs/developer/commits.md';

// See rule docs https://commitlint.js.org/#/reference-rules
const customRules = {
  'header-max-length': [2, 'always', 100],
  'body-leading-blank': [2, 'always'],
  'footer-leading-blank': [2, 'always'],
  'subject-case': [0],
  'body-max-line-length': [1, 'always', 100],
};

async function getCommitsInMr() {
  const diffBaseSha = CI_MERGE_REQUEST_DIFF_BASE_SHA;
  const sourceBranchSha = LAST_MR_COMMIT;
  const messages = await read({ from: diffBaseSha, to: sourceBranchSha });
  return messages;
}

const messageMatcher = r => r.test.bind(r);

async function isConventional(message) {
  return lint(
    message,
    { ...config.rules, ...customRules },
    {
      defaultIgnores: false,
      ignores: [
        messageMatcher(/^[Rr]evert .*/),
        messageMatcher(/^(?:fixup|squash)!/),
        messageMatcher(/^Merge branch/),
        messageMatcher(/^\d+\.\d+\.\d+/),
      ],
    },
  );
}

async function lintMr() {
  const commits = await getCommitsInMr();

  // When MR is set to squash, but it's not yet being merged, we check the MR Title
  if (
    CI_MERGE_REQUEST_SQUASH_ON_MERGE === 'true' &&
    CI_MERGE_REQUEST_EVENT_TYPE !== 'merge_train'
  ) {
    console.log(
      'INFO: The MR is set to squash. We will lint the MR Title (used as the commit message by default).',
    );
    return isConventional(CI_MERGE_REQUEST_TITLE).then(Array.of);
  }
  console.log('INFO: Checking all commits that will be added by this MR.');
  return Promise.all(commits.map(commit => isConventional(commit)));
}

async function run() {
  if (!CI) {
    console.error('This script can only run in GitLab CI.');
    process.exit(1);
  }

  if (!LAST_MR_COMMIT) {
    console.error(
      'LAST_MR_COMMIT environment variable is not present. Make sure this script is run from `lint.sh`',
    );
    process.exit(1);
  }

  const results = await lintMr();

  console.error(format({ results }, { helpUrl: urlSemanticRelease }));

  const numOfErrors = results.reduce((acc, result) => acc + result.errors.length, 0);
  if (numOfErrors !== 0) {
    process.exit(1);
  }
}

run().catch(err => {
  console.error(err);
  process.exit(1);
});
