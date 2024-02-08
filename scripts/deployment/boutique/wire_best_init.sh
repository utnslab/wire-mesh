#!/usr/bin/env bash
# Initialize boutique application for the Wire service mesh.

: "${TESTBED:=$HOME}"

# First start all kubernetes services
pushd $TESTBED/scripts
kubectl apply -f deployment/boutique/yaml/wire-best-kube.yaml
popd

# Use wire with istio injection only where needed
pushd $TESTBED/istio-1.16.1

# Get Istio inject configurations
kubectl -n istio-system get configmap istio-sidecar-injector -o=jsonpath='{.data.config}' > inject-config.yaml
kubectl -n istio-system get configmap istio-sidecar-injector -o=jsonpath='{.data.values}' > inject-values.yaml
kubectl -n istio-system get configmap istio -o=jsonpath='{.data.mesh}' > mesh-config.yaml

./bin/istioctl kube-inject \
    --injectConfigFile inject-config.yaml \
    --meshConfigFile mesh-config.yaml \
    --valuesFile inject-values.yaml \
    --filename $TESTBED/scripts/deployment/boutique/yaml/wire-best-istio.yaml \
    | kubectl apply -f -

popd
