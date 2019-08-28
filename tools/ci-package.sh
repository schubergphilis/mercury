#!/bin/bash -e

#REF=$(git log --graph  --pretty=format:%D -1 | cut -f2 -d, | sed -e 's/.*\///g')
#echo "ref: ${REF}"
git describe --tags --always > .version
echo "path: ${PWD} version: $(cat .version)"

if [ "${CIRCLE_BRANCH}" != "master" ]; then
    echo "Branch is: [${CIRCLE_BRANCH}] skipping packaging"
    exit 0
fi

if [ "$1" == "" ]; then
    REF=$(git log -1 --pretty=%B)
    echo "Last commit subject: ${REF}"
else
    REF="$@"
    echo "Manual commit subject: ${REF}"
fi


major=$(cat .version | cut -f1 -d.)
minor=$(cat .version | cut -f2 -d.)
patch=$(cat .version | cut -f3 -d. | cut -f1 -d-)
oldversion="${major}.${minor}.${patch}"
case "${REF}" in
    bugfix:*|bug:*|fix:*|automatic-patch:*)
        patch=$((patch+1))
        ;;
    feature:*|feat:*)
        patch=0
        minor=$((minor+1))
        ;;
    major:*)
        patch=0
        minor=0
        major=$((major+1))
        ;;
esac
newversion="${major}.${minor}.${patch}"

if [ "${oldversion}" == "${newversion}" ]; then
    echo "version not updated: old: ${oldversion} new: ${newversion}"
    exit 0
fi

echo "new version to be created: old: ${oldversion} new: ${newversion}"
echo "${newversion}" > .version

sudo apt-get --no-install-recommends install ruby ruby-dev rubygems build-essential rpm
sudo gem install --no-ri --no-rdoc fpm

make linux-package

go get github.com/tcnksm/ghr
VERSION=$(cat .version)

# add go version to bins
go version > ./build/packages/golang.version

ghr -soft -t ${GITHUB_TOKEN} -u ${CIRCLE_PROJECT_USERNAME} -r ${CIRCLE_PROJECT_REPONAME} -c ${CIRCLE_SHA1} -n "${CIRCLE_PROJECT_REPONAME^} v${VERSION}" ${VERSION} ./build/packages/
