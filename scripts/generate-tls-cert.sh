#!/usr/bin/env bash

#   Copyright Docker attest-provider authors

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


set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
cd "${REPO_ROOT}" || exit 1
NAMESPACE=security

generate() {
    # generate CA key and certificate
    echo "Generating CA key and certificate for attest-provider..."
    openssl genrsa -out ca.key 2048
    openssl req -new -x509 -days 3650 -key ca.key -subj "/O=Gatekeeper/CN=Gatekeeper Root CA" -out ca.crt

    # generate server key and certificate
    echo "Generating server key and certificate for attest-provider..."
    openssl genrsa -out tls.key 2048
    openssl req -newkey rsa:2048 -nodes -keyout tls.key -subj "/CN=attest-provider.${NAMESPACE}" -out server.csr
    openssl x509 -req -extfile <(printf "subjectAltName=DNS:attest-provider.%s" "${NAMESPACE}") -days 3650 -sha256 -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out tls.crt
}

mkdir -p "${REPO_ROOT}/certs"
pushd "${REPO_ROOT}/certs"
generate
popd
