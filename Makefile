#
# Makefile
# @author Ronald Doorn <rdoorn@schubergphilis.com>
#

.PHONY: update clean build build-all run package deploy test authors dist

export PATH := $(PATH):$(GOPATH)/bin
# Since go 1.13 GO111MODULE can be used even in GOPATH
export GO111MODULE=auto

NAME := mercury
VERSION := $(shell [ -f .version ] && cat .version || echo "pipeline-test")
GITTAG := $(shell git describe --tags --always --abbrev=0)
LASTCOMMIT := $(shell git rev-parse --verify HEAD)
BUILD := $(shell cat tools/rpm/BUILDNR)
LDFLAGS := "-X main.version=$(VERSION) -X main.versionBuild=$(BUILD) -X main.versionSha=$(LASTCOMMIT)"
PENDINGCOMMIT := $(shell git diff-files --quiet --ignore-submodules && echo 0 || echo 1)
LOCALIP := $(shell ifconfig | grep "inet " | grep broadcast | awk {'print $$2'} | head -1 )
GODIRS := $(shell go list -f '{{.Dir}}' ./...)

default: build

clean:
	@echo Cleaning up...
	@rm -f build
	@echo Done.

rice:
	@echo Merging static content...
	@which rice >/dev/null; if [ $$? -eq 1 ]; then \
		go get github.com/GeertJohan/go.rice/rice; \
	fi;
	cd internal/core && rice embed-go
	@echo Done

builddir:
	@mkdir -p ./build/osx/
	@mkdir -p ./build/linux/
	@mkdir -p ./build/packages/$(NAME)/

osx: builddir rice
	@echo Building OSX...
	GOOS=darwin GOARCH=amd64 go build -v -o ./build/osx/$(NAME) -ldflags $(LDFLAGS) ./cmd/mercury
	@echo Done.

osx-fast: builddir
	@echo Building OSX skipping rice...
	GOOS=darwin GOARCH=amd64 go build -v -o ./build/osx/$(NAME) -ldflags $(LDFLAGS) ./cmd/mercury
	@echo Done.

osx-race: builddir rice
	@echo Building OSX...
	GOOS=darwin GOARCH=amd64 go build -race -v -o ./build/osx/$(NAME) -ldflags $(LDFLAGS) ./cmd/mercury
	@echo Done.

osx-static:
	@echo Building OSX...
	GOOS=darwin GOARCH=amd64 go build -v -o ./build/osx/$(NAME) -ldflags '-s -w --extldflags "-static”  $(LDFLAGS)' ./cmd/mercury
	@echo Done.

linux: builddir rice
	@echo Building Linux...
	GOOS=linux GOARCH=amd64 go build -v -o ./build/linux/$(NAME) -ldflags '-s -w --extldflags "-static”  $(LDFLAGS)' ./cmd/mercury
	@echo Done.

build: osx linux

makeconfig:
	@echo Making config...
	cat ./test/$(NAME)-template.toml | sed -e 's/%LOCALIP%/$(LOCALIP)/g' > ./test/$(NAME).toml
	cat ./test/$(NAME).toml | sed -e 's/port = 9/port = 10/g' -e 's/localhost1/localhost3/' -e 's/localhost2/localhost1/' -e 's/localhost3/localhost2/' -e 's/127.0.0.1:9000/127.0.0.1:8000/' -e 's/127.0.0.1:10000/127.0.0.1:9000/' -e 's/127.0.0.1:8000/127.0.0.1:10000/' -e 's/preference = 0#1/preference = 1/' -e 's/preference = 1#0/preference = 0/' -e 's/15353/25353/' -e 's/ip = "127.0.0.1"/ip = "127.0.0.2"/' > ./test/$(NAME)-secondary.toml

run: osx makeconfig
	./build/osx/$(NAME) --config-file ./test/$(NAME).toml --pid-file /tmp/mercury.pid

run-linux: linux makeconfig
	./build/linux/$(NAME) --config-file ./test/$(NAME).toml --pid-file /tmp/mercury.pid

run-race: osx-race makeconfig
	./build/osx/$(NAME) --config-file ./test/$(NAME).toml --pid-file /tmp/mercury.pid

run-secondary: makeconfig
	./build/osx/$(NAME) --config-file ./test/$(NAME)-secondary.toml --pid-file /tmp/mercury-secondary.pid

run-noconfig: osx
	./build/osx/$(NAME) --config-file ./test//$(NAME).toml --pid-file /tmp/mercury.pid

run-secondary-noconfig:
	./build/osx/$(NAME) --config-file ./test/$(NAME)-secondary.toml --pid-file /tmp/mercury-secondary.pid

sudo-run: osx
	sudo ./build/osx/$(NAME) --config-file ./test/$(NAME).toml --pid-file /tmp/mercury.pid

test:
	go test -v ./...
	go test -v ./... --race --short
	go vet ./...

coverage: ## Shows coverage
	@go tool cover 2>/dev/null; if [ $$? -eq 3 ]; then \
		go get -u golang.org/x/tools/cmd/cover; \
	fi
	./tools/coverage.sh

coverage-upload:
	curl -L https://codeclimate.com/downloads/test-reporter/test-reporter-latest-linux-amd64 > ./cc-test-reporter
	chmod +x ./cc-test-reporter
	./cc-test-reporter after-build
	rm -f ./cc-test-reporter

prep_package:
	gem install fpm

committed:
ifndef CIRCLECI
ifeq ($(PENDINGCOMMIT), 1)
	git diff
	$(error You have a pending commit, please commit your code before making a package $(PENDINGCOMMIT))
endif
endif

linux-package: builddir linux committed
	mkdir -p ./build/packages/$(NAME)/
	cp -a ./tools/rpm/$(NAME)/* ./build/packages/$(NAME)/
	cp ./build/linux/$(NAME) ./build/packages/$(NAME)/usr/sbin/
	cp ./tools/html/* ./build/packages/$(NAME)/var/$(NAME)/
	fpm -s dir -t rpm -C ./build/packages/$(NAME) --name $(NAME) --rpm-os linux --version $(VERSION) --iteration $(BUILD) --exclude "*/.keepme"
	rm -rf ./build/packages/$(NAME)/
	mv $(NAME)-$(VERSION)*.rpm build/packages/

docker-alpine:
	cd docker && docker build --no-cache -t mercury-alpine:$(GITTAG) . -f Dockerfile.alpine

docker-scratch:
	cd docker && docker build --no-cache -t mercury:$(GITTAG) . -f Dockerfile.scratch

docker: docker-scratch

docker-prep:
	docker login

docker-upload-scratch: docker-scratch docker-prep
	$(eval DOCKERTAG = $(shell docker images mercury:$(GITTAG) --format "{{.ID}}"))
	echo "tag: $(DOCKERTAG)"
	docker tag $(DOCKERTAG) rdoorn/mercury:$(GITTAG)
	docker push rdoorn/mercury:$(GITTAG)
	docker tag rdoorn/mercury:$(GITTAG) rdoorn/mercury:latest
	docker push rdoorn/mercury:latest

docker-upload-alpine: docker-alpine docker-prep
	$(eval DOCKERTAG = $(shell docker images mercury-alpine:$(GITTAG) --format "{{.ID}}"))
	echo "tag: $(DOCKERTAG)"
	docker tag $(DOCKERTAG) rdoorn/mercury-alpine:$(GITTAG)
	docker push rdoorn/mercury-alpine:$(GITTAG)
	docker tag rdoorn/mercury-alpine:$(GITTAG) rdoorn/mercury-alpine:latest
	docker push rdoorn/mercury-alpine:latest

deps: ## Updates the vendored Go dependencies
	go mod download
	go mod vendor

updatedeps: ## Updates the vendored Go dependencies
	go get -u ./...
	go mod vendor

#authors:
#	@git log --format='%aN <%aE>' | LC_ALL=C.UTF-8 sort | uniq -c | sort -nr | sed "s/^ *[0-9]* //g" > AUTHORS
#	@cat AUTHORS
#
