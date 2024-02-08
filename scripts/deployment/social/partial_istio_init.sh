#!/usr/bin/env bash
# Initialize Istio for partial set of services.
# Args: Array of services to initialize Istio for.

: "${TESTBED:=$HOME}"

if [ "$#" -lt 1 ]; then
  echo "Usage: $0 <services>"
  exit 1
fi

SERVICES=("$@")
ALL_SERVICES=("compose-post-service" "home-timeline-redis" "home-timeline-service" "jaeger" "media-frontend" "media-memcached" "media-mongodb" "media-service" "nginx-thrift" "post-storage-memcached" "post-storage-mongodb" "post-storage-service" "social-graph-mongodb" "social-graph-redis" "social-graph-service" "text-service" "unique-id-service" "url-shorten-memcached" "url-shorten-mongodb" "url-shorten-service" "user-memcached" "user-mention-service" "user-mongodb" "user-service" "user-timeline-mongodb" "user-timeline-redis" "user-timeline-service")

pushd $TESTBED/DeathStarBench/socialNetwork/kubernetes
for service in "${ALL_SERVICES[@]}"; do
  name=$service
  # For service not in SERVICES, apply kube manifests
  if [[ ! " ${SERVICES[@]} " =~ " ${name} " ]]; then
    kubectl apply -Rf $service/
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
  for file in $TESTBED/DeathStarBench/socialNetwork/kubernetes/$service/*; do
    filename=$(basename "$file")

    # Use Istio for only service and deployment files, use kubectl for the rest
    ./bin/istioctl kube-inject \
    --injectConfigFile inject-config.yaml \
    --meshConfigFile mesh-config.yaml \
    --valuesFile inject-values.yaml \
    --filename $file \
    | kubectl apply -f -
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
