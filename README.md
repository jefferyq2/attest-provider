# Attest External Data Provider

OPA Gatekeeper external data provider implementation for Docker attest library image attestation verification.

## Prerequisites

- [ ] [`docker`](https://docs.docker.com/get-docker/)
- [ ] [`helm`](https://helm.sh/)
- [ ] [`kind`](https://kind.sigs.k8s.io/)
- [ ] [`kubectl`](https://kubernetes.io/docs/tasks/tools/#kubectl)

## Quick Start

1. Create a [kind cluster](https://kind.sigs.k8s.io/docs/user/quick-start/).

```bash
kind create cluster --name gatekeeper
```

2. Install the latest version of Gatekeeper and enable the external data feature.

```bash
# Add the Gatekeeper Helm repository
helm repo add gatekeeper https://open-policy-agent.github.io/gatekeeper/charts

# Install the latest version of Gatekeeper with the external data feature enabled.
helm install gatekeeper/gatekeeper \
    --set enableExternalData=true \
    --set validatingWebhookFailurePolicy=Fail \
    --set validatingWebhookTimeoutSeconds=30 \
    --set postInstall.probeWebhook.enabled=false \
    --set postInstall.labelNamespace.enabled=false \
    --name-template=gatekeeper \
    --namespace security \
    --create-namespace
```

3. Build and deploy the external data provider.

```bash
git clone https://github.com/docker/attest-external-data-provider.git
cd attest-external-data-provider

# if you are not planning to establish mTLS between the provider and Gatekeeper,
# deploy the provider to a separate namespace. Otherwise, do not run the following command
# and deploy the provider to the same namespace as Gatekeeper.
export NAMESPACE=security

# generate a self-signed certificate for the external data provider
./scripts/generate-tls-cert.sh

# build the image via docker buildx
make docker-buildx

# load the image into kind
make kind-load-image

# Choose one of the following ways to deploy the external data provider:

# 1. client and server auth enabled (recommended)
helm install attest-provider charts/external-data-provider \
    --set provider.tls.caBundle="$(cat certs/ca.crt | base64 | tr -d '\n\r')" \
    --namespace "${NAMESPACE:-gatekeeper-system}"

# 2. client auth disabled and server auth enabled
helm install attest-provider charts/external-data-provider \
    --set clientCAFile="" \
    --set provider.tls.caBundle="$(cat certs/ca.crt | base64 | tr -d '\n\r')" \
    --namespace "${NAMESPACE:-gatekeeper-system}" \
    --create-namespace
```

4. Install constraint template and constraint.

```bash
kubectl apply -f validation/attest-constraint-template.yaml
kubectl apply -f validation/attest-constraint.yaml
```

5. Test the external data provider by dry-running the following command:

```bash
kubectl create ns test
kubectl run nginx --image nginx -n test --dry-run=server -ojson
```

Gatekeeper should deny the pod admission above because the image `nginx` is missing signed annotations but has an image policy in tuf-staging.

TODO: implement mutating policy (tag -> digest)

<!-- 6. Install Assign mutation.

```bash
kubectl apply -f mutation/external-data-provider-mutation.yaml
```

7. Test the external data provider by dry-running the following command:

```bash
kubectl run nginx --image=nginx --dry-run=server -ojson
```

The expected JSON output should have the following image field with `_valid` appended by the external data provider:

```json
"containers": [
    {
        "name": "nginx",
        "image": "nginx_valid",
        ...
    }
]
``` -->

1. To reload the attest-provider image after making changes, run the following command:

```bash
make reload
```

1. Uninstall the external data provider and Gatekeeper.

```bash
kubectl delete -f validation/
# kubectl delete -f mutation/ TODO: implement mutation
helm uninstall attest-provider --namespace "${NAMESPACE:-gatekeeper-system}"
helm uninstall gatekeeper --namespace security
```
