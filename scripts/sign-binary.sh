#!/bin/bash
set -e

os="$1"
artifact="$2"

if [ -z "$os" ] || [ -z "$artifact" ]; then
	echo "Usage: $0 <macos|windows> <artifact_path>" >&2
	exit 1
fi

echo "Processing artifact: $artifact (OS: $os)"

if [ -z "$CI" ] || [ -z "$CI_COMMIT_TAG" ]; then
  echo "Skipping code signing: not in CI release environment (CI=$CI, CI_COMMIT_TAG=$CI_COMMIT_TAG)"
  exit 0
fi

# Verify the file exists in the container
if [ ! -f "$artifact" ]; then
	echo "ERROR: Artifact file does not exist: $artifact" >&2
	exit 1
fi

# Validate required environment variables
if [ -z "$GCLOUD_PROJECT" ] || [ -z "$GOOGLE_APPLICATION_CREDENTIALS" ] || [ -z "$HOST_PWD" ]; then
	echo "Error: Missing required environment variables" >&2
	echo "Required: GCLOUD_PROJECT, GOOGLE_APPLICATION_CREDENTIALS, HOST_PWD" >&2
	exit 1
fi

# Getting the artifact filename relative to the code-signer image is a bit tricky.
# For example, let's consider the absolute path inside the GoRelease container is
# `/go/src/gitlab.com/gitlab-org/cli/dist/windows_windows_amd64_v1/bin/glab.exe`.
#
# 1. First, get the working directory inside the GoReleaser container (/go/src/gitlab.com/gitlab-org/cli).
# 2. Strip it from the artifact path to get the relative path (dist/windows_windows_386_sse2/bin/glab.exe).
# 3. Mount the host directory (${HOST_PWD}) to /work in the code-signer container.
# 4. Access the file as /work/dist/windows_windows_386_sse2/bin/glab.exe.
WORK_DIR="$(pwd)"
echo "Working directory: $WORK_DIR"
echo "Artifact absolute path: $artifact"
echo "HOST_PWD: $HOST_PWD"
artifact_relative="${artifact#"${WORK_DIR}"/}"

# On the host, the file will be at: ${HOST_PWD}/${artifact_relative}
echo "Relative path: $artifact_relative"

# Sign based on OS
if [ "$os" = "windows" ]; then
	echo "Signing Windows binary: $artifact"
	docker run \
		--rm \
		-e "GCLOUD_PROJECT=${GCLOUD_PROJECT}" \
		-e "GOOGLE_APPLICATION_CREDENTIALS=${GOOGLE_APPLICATION_CREDENTIALS}" \
		-v "${HOST_PWD}/.gitlab-secrets:/var/run/secrets/gitlab" \
		-v "${HOST_PWD}:/work" \
		registry.gitlab.com/gitlab-com/gl-infra/common-ci-tasks-images/code-signer:1.3.0 \
		sign-windows-binaries --overwrite "/work/${artifact_relative}"

elif [ "$os" = "macos" ]; then
	if [ -z "$APPSTORE_CONNECT_API_KEY_FILE" ]; then
		echo "Error: APPSTORE_CONNECT_API_KEY_FILE is required for macOS signing" >&2
		exit 1
	fi

	echo "Signing macOS binary: $artifact"
	docker run \
		--rm \
		-e "APPSTORE_CONNECT_API_KEY_FILE=${APPSTORE_CONNECT_API_KEY_FILE}" \
		-e "GCLOUD_PROJECT=${GCLOUD_PROJECT}" \
		-e "GOOGLE_APPLICATION_CREDENTIALS=${GOOGLE_APPLICATION_CREDENTIALS}" \
		-v "${HOST_PWD}/.gitlab-secrets:/var/run/secrets/gitlab" \
		-v "${HOST_PWD}:/work" \
		-v "${APPSTORE_CONNECT_API_KEY_FILE}:${APPSTORE_CONNECT_API_KEY_FILE}" \
		registry.gitlab.com/gitlab-com/gl-infra/common-ci-tasks-images/code-signer:1.3.0 \
		sign-macos-binaries --overwrite "/work/${artifact_relative}"

else
	echo "Not a Windows or macOS binary, no code signing needed for: $os"
	exit 0
fi

echo "Produced the following artifacts after signing:"
ls -al "$artifact"
