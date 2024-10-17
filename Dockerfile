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
ARG BUILDERIMAGE="golang:1.22"
ARG BASEIMAGE="gcr.io/distroless/static:nonroot"

FROM ${BUILDERIMAGE} AS builder

ENV CGO_ENABLED=0

WORKDIR /app

# --- This block can be removed when github.com/docker/attest is public
ENV GOPRIVATE="github.com/docker/attest"
RUN --mount=type=secret,id=GITHUB_TOKEN <<EOT
  set -e
  GITHUB_TOKEN=${GITHUB_TOKEN:-$(cat /run/secrets/GITHUB_TOKEN)}
  if [ -n "$GITHUB_TOKEN" ]; then
    echo "Setting GitHub access token"
    git config --global "url.https://x-access-token:${GITHUB_TOKEN}@github.com.insteadof" "https://github.com"
  fi
EOT
# ---

ARG VERSION="dev"

RUN --mount=type=bind,source=.,target=/app \
    --mount=type=cache,target=$GOPATH/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags "-X main.version=$VERSION" -o /bin/attest main.go

FROM ${BASEIMAGE} AS production

COPY --from=builder /bin/attest /

USER 65532:65532

ENTRYPOINT ["/attest"]

FROM production AS dev

COPY --chown=65532:65532 certs/tls.crt certs/tls.key /certs/
