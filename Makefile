#
# Makefile
# @author Ronald Doorn <rdoorn@schubergphilis.com>
#

.PHONY: update clean build build-all run package deploy test authors dist get

#export GOPATH := ${PWD}/vendor:${PWD}
#export GOBIN := ${PWD}/vendor/bin


NAME := mercury
VERSION := $(shell cat VERSION)
LASTCOMMIT := $(shell git rev-parse --verify HEAD)
BUILD := $(shell cat build/linux/BUILDNR)
LDFLAGS := "-X main.version=${VERSION} -X main.versionBuild=${BUILD} -X main.versionSha=${LASTCOMMIT}"
#sGOPATH := "${PWD}/vendor:${PWD}"
#PENDINGCOMMIT := "$(git diff-files --quiet --ignore-submodules && echo 0 || echo 1 )"
#PENDINGCOMMIT := $(git diff-files --quiet --ignore-submodules && echo 1 || echo 0 )
PENDINGCOMMIT := $(shell git diff-files --quiet --ignore-submodules && echo 0 || echo 1)
LOCALIP := $(shell ifconfig | grep "inet " | grep broadcast | awk {'print $$2'} )

default: build

clean:
	@echo Cleaning up...
	@rm -f bin/*/*
	@echo Done.

get:
	@echo Getting...
	go get ./src/
	@echo Done

rice:
	@echo Merging static content...
	if [ -a ${GOPATH}/rice ] ; \
	then \
	go get github.com/GeertJohan/go.rice/rice; \
	fi;
	cd src/core && rice embed-go
	@echo Done

osx: rice get
	@echo Building OSX...
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -v -o ./bin/osx/$(NAME) -ldflags ${LDFLAGS} ./src/*.go
	@echo Done.

osx-fast:
	@echo Building OSX skipping rice and get...
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -v -o ./bin/osx/$(NAME) -ldflags ${LDFLAGS} ./src/*.go
	@echo Done.

osx-race: rice get
	@echo Building OSX...
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -race -v -o ./bin/osx/$(NAME) -ldflags ${LDFLAGS} ./src/*.go
	@echo Done.

linux: rice get
	@echo Building Linux...
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -v -o ./bin/linux/$(NAME) -ldflags '-s -w --extldflags "-static”  ${LDFLAGS}' ./src/*.go
	@echo Done.

linux-copy: linux
	@echo Copy Linux...
	scp ./bin/linux/$(NAME) fxtplb001:
	@echo Done.

build: osx linux

build-static:
	@echo Building...
	CGO_ENABLED=0 go build -v -o ./bin/$(NAME) -ldflags '-s -w --extldflags "-static”  ${LDFLAGS}' ./src/*.go
	@echo Done.

makeconfig:
	@echo Making config...
	cat ./build/test/${NAME}-template.toml | sed -e 's/%LOCALIP%/${LOCALIP}/g' > ./build/test/${NAME}.toml
	cat ./build/test/${NAME}.toml | sed -e 's/port = 9/port = 10/g' -e 's/localhost1/localhost3/' -e 's/localhost2/localhost1/' -e 's/localhost3/localhost2/' -e 's/127.0.0.1:9000/127.0.0.1:8000/' -e 's/127.0.0.1:10000/127.0.0.1:9000/' -e 's/127.0.0.1:8000/127.0.0.1:10000/' > ./build/test/${NAME}-secondary.toml

run: osx makeconfig
	./bin/osx/$(NAME) --config-file ./build/test//${NAME}.toml --pid-file /tmp/mercury.pid

run-race: osx-race makeconfig
	./bin/osx/$(NAME) --config-file ./build/test/${NAME}.toml --pid-file /tmp/mercury.pid

run-secondary: makeconfig
	./bin/osx/$(NAME) --config-file ./build/test/${NAME}-secondary.toml --pid-file /tmp/mercury-secondary.pid

run-noconfig: osx
	./bin/osx/$(NAME) --config-file ./build/test//${NAME}.toml --pid-file /tmp/mercury.pid

run-secondary-noconfig:
	./bin/osx/$(NAME) --config-file ./build/test/${NAME}-secondary.toml --pid-file /tmp/mercury-secondary.pid

sudo-run: osx
	sudo ./bin/osx/$(NAME) --config-file ./build/test/${NAME}.toml --pid-file /tmp/mercury.pid

test:
	@go test -v ./src/config/*.go --config-file ../../build/test/${NAME}.toml

cover: ## Shows coverage
	@go tool cover 2>/dev/null; if [ $$? -eq 3 ]; then \
		go get -u golang.org/x/tools/cmd/cover; \
	fi
	go test ./src/config -coverprofile=coverage.out
	go tool cover -html=coverage.out
	rm coverage.out

prep_package:
	gem install fpm

committed:
ifeq ($(PENDINGCOMMIT), 1)
	   $(error You have a pending commit, please commit your code before making a package ${PENDINGCOMMIT})
endif

linux-package: linux committed
	cp ./bin/linux/$(NAME) ./build/linux/$(NAME)/usr/sbin/
	mkdir -p ./build/linux/$(NAME)/var/$(NAME)/
	cp ./build/html/* ./build/linux/$(NAME)/var/$(NAME)/
	fpm -s dir -t rpm -C ./build/linux/$(NAME) --name $(NAME) --rpm-os linux --version ${VERSION} --iteration ${BUILD} --exclude "*/.keepme"
	mv $(NAME)-${VERSION}*.rpm build/packages/
	awk '{$$1=$$1+1}1' build/linux/BUILDNR  > build/linux/BUILDNR.tmp && mv build/linux/BUILDNR.tmp build/linux/BUILDNR

docker-scratch:
	if [ -a /System/Library/Keychains/SystemRootCertificates.keychain ] ; \
	then \
		security find-certificate /System/Library/Keychains/SystemRootCertificates.keychain > build/docker/ca-certificates.crt; \
	fi;
	if [ -a /etc/ssl/certs/ca-certificates.crt ] ; \
	then \
		cp /etc/ssl/certs/ca-certificates.crt build/docker/ca-certificates.crt; \
	fi;
	docker build -t mercury-scratch -f build/docker/Dockerfile.scratch .

#authors:
#	@git log --format='%aN <%aE>' | LC_ALL=C.UTF-8 sort | uniq -c | sort -nr | sed "s/^ *[0-9]* //g" > AUTHORS
#	@cat AUTHORS
#
