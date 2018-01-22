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
BUILD := $(shell cat tools/rpm/BUILDNR)
LDFLAGS := "-X main.version=${VERSION} -X main.versionBuild=${BUILD} -X main.versionSha=${LASTCOMMIT}"
#sGOPATH := "${PWD}/vendor:${PWD}"
#PENDINGCOMMIT := "$(git diff-files --quiet --ignore-submodules && echo 0 || echo 1 )"
#PENDINGCOMMIT := $(git diff-files --quiet --ignore-submodules && echo 1 || echo 0 )
PENDINGCOMMIT := $(shell git diff-files --quiet --ignore-submodules && echo 0 || echo 1)
LOCALIP := $(shell ifconfig | grep "inet " | grep broadcast | awk {'print $$2'} )

default: build

clean:
	@echo Cleaning up...
	@rm -f build/linux/*
	@rm -f build/osx/*
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

builddir:
	@mkdir -p ./build/osx/
	@mkdir -p ./build/linux/
	@mkdir -p ./build/packages/

osx: builddir rice get
	@echo Building OSX...
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -v -o ./build/osx/$(NAME) -ldflags ${LDFLAGS} ./src/*.go
	@echo Done.

osx-fast: builddir
	@echo Building OSX skipping rice and get...
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -v -o ./build/osx/$(NAME) -ldflags ${LDFLAGS} ./src/*.go
	@echo Done.

osx-race: builddir rice get
	@echo Building OSX...
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -race -v -o ./build/osx/$(NAME) -ldflags ${LDFLAGS} ./src/*.go
	@echo Done.

osx-static:
	@echo Building...
	CGO_ENABLED=0 go build -v -o ./build/osx/$(NAME) -ldflags '-s -w --extldflags "-static”  ${LDFLAGS}' ./src/*.go
	@echo Done.

linux: builddir rice get
	@echo Building Linux...
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -v -o ./build/linux/$(NAME) -ldflags '-s -w --extldflags "-static”  ${LDFLAGS}' ./src/*.go
	@echo Done.

build: osx linux

makeconfig:
	@echo Making config...
	cat ./test/${NAME}-template.toml | sed -e 's/%LOCALIP%/${LOCALIP}/g' > ./test/${NAME}.toml
	cat ./test/${NAME}.toml | sed -e 's/port = 9/port = 10/g' -e 's/localhost1/localhost3/' -e 's/localhost2/localhost1/' -e 's/localhost3/localhost2/' -e 's/127.0.0.1:9000/127.0.0.1:8000/' -e 's/127.0.0.1:10000/127.0.0.1:9000/' -e 's/127.0.0.1:8000/127.0.0.1:10000/' > ./test/${NAME}-secondary.toml

run: osx makeconfig
	./build/osx/$(NAME) --config-file ./test/${NAME}.toml --pid-file /tmp/mercury.pid

run-race: osx-race makeconfig
	./build/osx/$(NAME) --config-file ./test/${NAME}.toml --pid-file /tmp/mercury.pid

run-secondary: makeconfig
	./build/osx/$(NAME) --config-file ./test/${NAME}-secondary.toml --pid-file /tmp/mercury-secondary.pid

run-noconfig: osx
	./build/osx/$(NAME) --config-file ./test//${NAME}.toml --pid-file /tmp/mercury.pid

run-secondary-noconfig:
	./build/osx/$(NAME) --config-file ./test/${NAME}-secondary.toml --pid-file /tmp/mercury-secondary.pid

sudo-run: osx
	sudo ./build/osx/$(NAME) --config-file ./test/${NAME}.toml --pid-file /tmp/mercury.pid

test:
	@go test -v ./src/config/*.go --config-file ../../test/${NAME}.toml

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

linux-package: builddir linux committed
	mkdir -p ./build/packages/$(NAME)/usr/sbin/
	mkdir -p ./build/packages/$(NAME)/var/$(NAME)/
	cp ./build/linux/$(NAME) ./build/packages/$(NAME)/usr/sbin/
	cp ./tools/html/* ./build/packages/$(NAME)/var/$(NAME)/
	fpm -s dir -t rpm -C ./build/packages/$(NAME) --name $(NAME) --rpm-os linux --version ${VERSION} --iteration ${BUILD} --exclude "*/.keepme"
	mv $(NAME)-${VERSION}*.rpm build/packages/
	awk '{$$1=$$1+1}1' tools/rpm/BUILDNR  > tools/rpm/BUILDNR.tmp && mv tools/rpm/BUILDNR.tmp tools/rpm/BUILDNR

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
