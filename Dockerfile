ARG BUILDERIMAGE="golang:1.22"
ARG BASEIMAGE="gcr.io/distroless/static:nonroot"

FROM ${BUILDERIMAGE} as builder

ARG LDFLAGS

ENV GO111MODULE=on \
  CGO_ENABLED=0

WORKDIR /go/src/github.com/docker/attest-external-data-provider

COPY . .

# --- This block can be replaced by `RUN go mod download` when github.com/docker/attest is public
ENV GOPRIVATE="github.com/docker/attest"
RUN --mount=type=secret,id=GITHUB_TOKEN <<EOT
  set -e
  GITHUB_TOKEN=${GITHUB_TOKEN:-$(cat /run/secrets/GITHUB_TOKEN)}
  if [ -n "$GITHUB_TOKEN" ]; then
    echo "Setting GitHub access token"
    git config --global "url.https://x-access-token:${GITHUB_TOKEN}@github.com.insteadof" "https://github.com"
  fi
  go mod download
EOT
# ---
RUN make build

FROM ${BASEIMAGE}

COPY --from=builder /go/src/github.com/docker/attest-external-data-provider/bin/attest /

COPY --from=builder --chown=65532:65532 /go/src/github.com/docker/attest-external-data-provider/certs/tls.crt \
  /go/src/github.com/docker/attest-external-data-provider/certs/tls.key \
  /certs/

USER 65532:65532

ENTRYPOINT ["/attest"]
