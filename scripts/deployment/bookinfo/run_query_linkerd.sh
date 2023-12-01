#!/usr/bin/env bash
# Run query for the BookInfo benchmark with Linkerd service mesh
# Arguments:
# 1: First time install, Init (1) else (0)

if [ $# -lt 1 ]; then
  echo 1>&2 "Not enough arguments"
  exit 2
fi

: "${TESTBED:=$HOME}"

if [[ "$1" -eq "1" ]]; then
  # Init
  pushd $TESTBED/scripts
  kubectl apply -f deployment/bookinfo/bookinfo.yaml
  popd

  # Wait for the pods to get running
  sleep 3m

  # Inject linkerd
  pushd $TESTBED/.linkerd2
  kubectl get deploy -o yaml \
      | ./bin/linkerd inject - \
      | kubectl apply -f -
  popd

  # Wait for the sidecars to get running
  sleep 2m

  # Add and configure ingress
  kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.5.1/deploy/static/provider/cloud/deploy.yaml
  kubectl wait --namespace ingress-nginx \
    --for=condition=ready pod \
    --selector=app.kubernetes.io/component=controller \
    --timeout=120s

  # Use Kubernetes Ingress NGINX
  kubectl apply -f deployment/bookinfo/bookinfo-ingress.yaml
fi

INGRESS_HOST=$(kubectl -n ingress-nginx get service ingress-nginx-controller -o jsonpath='{.spec.clusterIP}')
INGRESS_PORT=$(kubectl -n ingress-nginx get svc ingress-nginx-controller -o jsonpath='{.spec.ports[?(@.name=="http")].port}')

GATEWAY_URL=$INGRESS_HOST:$INGRESS_PORT

# Run queries to log timings
TIMEFORMAT=%R

# Warm-up
for i in $(seq 1 200); do
{ time curl -s -o /dev/null "http://$GATEWAY_URL/productpage"; } 2>&1
done

for i in $(seq 1 1000); do
{ time curl -s -o /dev/null "http://$GATEWAY_URL/productpage"; } >> ~/out/time_bookinfo_linkerd.run 2>&1
done