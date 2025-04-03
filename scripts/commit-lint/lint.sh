#!/usr/bin/env bash

set -e # Abort script at first error, when a command exits with non-zero status (except in until or while loops, if-tests, list constructs)
set -u # Attempt to use undefined variable outputs error message, and forces an exit
# set -x  # Similar to verbose mode (-v), but expands commands
set -o pipefail # Causes a pipeline to return the exit status of the last command in the pipe that returned a non-zero return value.

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

get_last_mr_commit() {
  if [ "$CI_MERGE_REQUEST_EVENT_TYPE" == "detached" ]; then
    echo "WARNING: This project doesn't have merge train enabled. This script can't guarantee that invalid commits will not be added."
    echo "Invalid commits can be added if the person merging the MR changes the commit message or if somebody toggles the 'squash on merge' button after this job run."
    LAST_MR_COMMIT="$(git rev-parse HEAD)"
    return
  fi

  FIRST_PARENT="$(git rev-parse HEAD^1)"
  SECOND_PARENT="$(git rev-parse HEAD^2)"
  # The following two ifs check which of the parent commits is coming from the MR
  # We use the fact that the other parent is always the target branch (usually main) HEAD SHA
  if [ "$FIRST_PARENT" == "$CI_MERGE_REQUEST_TARGET_BRANCH_SHA" ]; then
    LAST_MR_COMMIT="$SECOND_PARENT"
  fi

  if [ "$SECOND_PARENT" == "$CI_MERGE_REQUEST_TARGET_BRANCH_SHA" ]; then
    LAST_MR_COMMIT="$FIRST_PARENT"
  fi
}

get_last_mr_commit
export LAST_MR_COMMIT

node "$SCRIPT_DIR/lint.js"
