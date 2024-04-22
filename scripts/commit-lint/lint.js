import axios from "axios";
import read from "@commitlint/read";
import lint from "@commitlint/lint";
import format from "@commitlint/format";
import config from "@commitlint/config-conventional";

const maximumLineLength = 72;

// You can test the script by setting these environment variables
const {
  CI_MERGE_REQUEST_PROJECT_ID, // 5261717
  CI_MERGE_REQUEST_IID,
  CI_COMMIT_SHA,
  CI_MERGE_REQUEST_TARGET_BRANCH_NAME, // usually main
} = process.env;

const urlSemanticRelease =
  'https://gitlab.com/gitlab-org/cli/-/blob/main/CONTRIBUTING.md#commit-messages';

const customRules = {
  'header-max-length': [2, 'always', maximumLineLength],
  'body-leading-blank': [2, 'always'],
  'footer-leading-blank': [2, 'always'],
  'subject-case': [0],
};

async function getMr() {
  const result = await axios.get(
    `https://gitlab.com/api/v4/projects/${CI_MERGE_REQUEST_PROJECT_ID}/merge_requests/${CI_MERGE_REQUEST_IID}`,
  );
  const { title, squash } = result.data;
  return {
    title,
    squash,
  };
}

async function getCommitsInMr() {
  const targetBranch = CI_MERGE_REQUEST_TARGET_BRANCH_NAME;
  const sourceCommit = CI_COMMIT_SHA;
  const messages = await read({ from: targetBranch, to: sourceCommit });
  return messages;
}

async function isConventional(message) {
  return lint(message, { ...config.rules, ...customRules }, { defaultIgnores: true });
}

async function lintMr() {
  const mr = await getMr();
  const commits = await getCommitsInMr();

  if (!mr.squash || commits.length === 1) {
    console.log(
      "INFO: Either the merge request isn't set to squash commits, or contains only one commit. Every commit message must use conventional commits.\n" +
        'INFO: For help, read https://gitlab.com/gitlab-org/gitlab-vscode-extension/-/blob/main/docs/developer/commits.md',
    );
    return Promise.all(commits.map(isConventional));
  }

  console.log(
    'INFO: The merge request is set to both squash commits and use the merge request title for the squash commit.\n' +
      'INFO: If the merge request title is incorrect, fix the title and rerun this CI/CD job.\n',
  );
  return isConventional(mr.title).then(Array.of);
}

async function run() {
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
