#!make

ifdef OS
# Windows build
GOCMD=go
else
# Unix build
GOCMD=/usr/local/go/bin/go
endif

# Go parameters
GOLINTCMD=golangci-lint
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

PROJECT_PATH=github.com/kettari/location-bot
PROJECT_CMD ?= console
BINARY_PATH=bin/
BINARY_NAME=location_$(PROJECT_CMD)
include deploy_$(PROJECT_CMD).env

# Assign build version
BUILD_VERSION := $(shell git describe --tags --always --dirty)

.PHONY: commit build deploy

commit:
	git add .
	git commit -m "WIP"
	git push

# build:
#	$(GOBUILD) -o $(BINARY_PATH)$(BINARY_NAME) -v $(PROJECT_NAME)/cmd/$(PROJECT_CMD)

build:
	@echo ">> building docker container $(PROJECT_CMD)"
	docker build \
		-f Dockerfile-$(PROJECT_CMD) \
	    -t $(DOCKER_REGISTRY_PREFIX)/$(APP_NAME)-$(APP_ENV):$(BUILD_VERSION) \
	    -t $(DOCKER_REGISTRY_PREFIX)/$(APP_NAME)-$(APP_ENV):latest \
	    --build-arg PROJECT_CMD=$(PROJECT_CMD) \
	    .

deploy:
	@echo ">> building & deploying docker container $(PROJECT_CMD)"
	docker build \
		-f Dockerfile-$(PROJECT_CMD) \
		-t $(DOCKER_REGISTRY_PREFIX)/$(APP_NAME)-$(APP_ENV):$(BUILD_VERSION) \
		-t $(DOCKER_REGISTRY_PREFIX)/$(APP_NAME)-$(APP_ENV):latest \
		--build-arg PROJECT_CMD=$(PROJECT_CMD) \
		.
	docker push $(DOCKER_REGISTRY_PREFIX)/$(APP_NAME)-$(APP_ENV):$(BUILD_VERSION)
	docker push $(DOCKER_REGISTRY_PREFIX)/$(APP_NAME)-$(APP_ENV):latest

.PHONY: lint
lint:
	$(GOLINTCMD) run ./...
