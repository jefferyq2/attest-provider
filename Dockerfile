ARG BUILDERIMAGE="golang:1.22"
ARG BASEIMAGE="gcr.io/distroless/static:nonroot"

FROM ${BUILDERIMAGE} as builder

ARG LDFLAGS

ENV GO111MODULE=on \
  CGO_ENABLED=0

WORKDIR /src/attest-provider

COPY . .

# --- This block can be replaced by `RUN go mod download` when github.com/docker/attest is public
ENV GOPRIVATE="github.com/docker/attest"
RUN --mount=type=cache,target=$GOPATH/pkg/mod --mount=type=secret,id=GITHUB_TOKEN <<EOT
  set -e
  GITHUB_TOKEN=${GITHUB_TOKEN:-$(cat /run/secrets/GITHUB_TOKEN)}
  if [ -n "$GITHUB_TOKEN" ]; then
    echo "Setting GitHub access token"
    git config --global "url.https://x-access-token:${GITHUB_TOKEN}@github.com.insteadof" "https://github.com"
  fi
  go mod download
EOT
# ---

RUN --mount=type=cache,target=$GOPATH/pkg/mod --mount=type=cache,target=/root/.cache/go-build make build

FROM ${BASEIMAGE}

COPY --from=builder /src/attest-provider/bin/attest /

COPY --from=builder --chown=65532:65532 /src/attest-provider/certs/tls.crt \
  /src/attest-provider/certs/tls.key \
  /certs/

USER 65532:65532

ENTRYPOINT ["/attest"]
