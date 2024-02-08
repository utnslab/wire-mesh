#!/usr/bin/env bash
# Initialize Istio for partial set of services.
# Args: Array of services to initialize Istio for.

: "${TESTBED:=$HOME}"

if [ "$#" -lt 1 ]; then
  echo "Usage: $0 <services>"
  exit 1
fi

SERVICES=("$@")
ALL_SERVICES=("ads" "cart" "catalog" "checkout" "currency" "email" "frontend" "payment" "recommendation" "redis" "shipping")

pushd $TESTBED/scripts/deployment/boutique/yaml
for service in "${ALL_SERVICES[@]}"; do
  # For service not in SERVICES, apply kube manifests
  if [[ ! " ${SERVICES[@]} " =~ " ${service} " ]]; then
    kubectl apply -Rf $service.yaml
  fi
done
popd

pushd $TESTBED/istio-1.16.1
# Get Istio inject configurations
kubectl -n istio-system get configmap istio-sidecar-injector -o=jsonpath='{.data.config}' > inject-config.yaml
kubectl -n istio-system get configmap istio-sidecar-injector -o=jsonpath='{.data.values}' > inject-values.yaml
kubectl -n istio-system get configmap istio -o=jsonpath='{.data.mesh}' > mesh-config.yaml

# The provided services are all for Istio
for service in "${SERVICES[@]}"
do
  for file in $TESTBED/scripts/deployment/boutique/yaml/*; do
    filename=$(basename "$file")

    # Use Istio for only service and deployment files, use kubectl for the rest
    if [[ $filename == $service".yaml" ]]; then
      ./bin/istioctl kube-inject \
      --injectConfigFile inject-config.yaml \
      --meshConfigFile mesh-config.yaml \
      --valuesFile inject-values.yaml \
      --filename $file \
      | kubectl apply -f -
    fi
  done
done
popd

# Wait for 1 minute and remove pods with (Evicted|Unknown|Completed|Error) status
sleep 1m
kubectl get pods | grep Evicted | awk '{print $1}' | xargs kubectl delete pod
kubectl get pods | grep Unknown | awk '{print $1}' | xargs kubectl delete pod
kubectl get pods | grep Completed | awk '{print $1}' | xargs kubectl delete pod
kubectl get pods | grep Error | awk '{print $1}' | xargs kubectl delete pod

# Wait for another 1 minute
sleep 1m
