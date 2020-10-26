#!/bin/bash -ex

#PREVIOUS_VERSION=$(git describe --tags --always | cut -f1 -d-)
#PREVIOUS_VERSION=$(git tag --sort=committerdate | tail -1)
echo "getting latest versions:"
git tag --sort=committerdate | sort --version-sort

PREVIOUS_VERSION=$(git tag --sort=committerdate | sort --version-sort | tail -1)
echo "latest git tag: ${PREVIOUS_VERSION}"
echo "getting latest golang.version @ https://github.com/schubergphilis/mercury/releases/download/${PREVIOUS_VERSION}/golang.version""
RESULT=$(curl -L https://github.com/schubergphilis/mercury/releases/download/${PREVIOUS_VERSION}/golang.version -o golang.version -w "%{http_code}")
PREVIOUS_GOLANG_VERSION=$(cat golang.version)
echo "latest golang version: ${PREVIOUS_GOLANG_VERSION}"
CURRENT_GOLANG_VERSION=$(go version)
echo "image golang version: ${PREVIOUS_GOLANG_VERSION}"

if [ "${RESULT}" != "200" ]; then
    echo "get golang version returned status: ${RESULT}"
    echo "url: https://github.com/schubergphilis/mercury/releases/download/${PREVIOUS_VERSION}/golang.version"
    exit 0
fi

echo "old: [${PREVIOUS_GOLANG_VERSION}]"
echo "new: [${CURRENT_GOLANG_VERSION}]"

if [ "${CURRENT_GOLANG_VERSION}" == "${PREVIOUS_GOLANG_VERSION}" ]; then
    echo "up to date with latest golang version"
    exit 0
fi

echo "new golang version available, rebuilding"
./tools/ci-package.sh automatic-patch: golang version update to ${CURRENT_GOLANG_VERSION}
