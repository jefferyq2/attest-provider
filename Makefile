#   Copyright 2024 Docker attest-provider authors
#
#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at
#
#       http://www.apache.org/licenses/LICENSE-2.0
#
#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.
REPOSITORY ?= docker/attest-provider
IMG := $(REPOSITORY):dev

# When updating this, make sure to update the corresponding action in
# workflow.yaml
GOLANGCI_LINT_VERSION := v1.50.0

# Detects the location of the user golangci-lint cache.
GOLANGCI_LINT_CACHE := $(shell pwd)/.tmp/golangci-lint

.PHONY: build
build:
	go build -o bin/attest main.go

# lint runs a dockerized golangci-lint, and should give consistent results
# across systems.
# Source: https://golangci-lint.run/usage/install/#docker
.PHONY: lint
lint:
	docker run --rm -v $(shell pwd):/app \
		-v ${GOLANGCI_LINT_CACHE}:/root/.cache/golangci-lint \
		-w /app golangci/golangci-lint:${GOLANGCI_LINT_VERSION}-alpine \
		golangci-lint run -v

.PHONY: docker-buildx-builder
docker-buildx-builder:
	if ! docker buildx ls | grep -q container-builder; then\
		docker buildx create --name container-builder --use;\
	fi

.PHONY: docker-buildx
docker-buildx: docker-buildx-builder
	docker buildx build --platform linux/amd64 --load -t ${IMG} . --secret=id=GITHUB_TOKEN

.PHONY: kind-load-image
kind-load-image:
	kind load docker-image ${IMG} --name gatekeeper

.PHONY: rollout-restart
rollout-restart:
	kubectl -n security rollout restart deployment/attest-provider

.PHONY: reload
reload: docker-buildx kind-load-image rollout-restart
