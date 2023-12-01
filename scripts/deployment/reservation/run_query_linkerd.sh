#!/usr/bin/env bash
# Run query for the Hotel Reservation benchmark with Istio service mesh
# Arguments:
# 1: First time install, Init (1) else (0)

if [ $# -lt 1 ]; then
  echo 1>&2 "Not enough arguments"
  exit 2
fi

: "${TESTBED:=$HOME}"
pushd $TESTBED

if [[ "$1" -eq "1" ]]; then
  sudo apt install -y luarocks
  sudo luarocks install luasocket

  # Pull docker image 
  docker pull divyanshus/hotelreservation
  
  git clone https://github.com/delimitrou/DeathStarBench.git
  pushd DeathStarBench/hotelReservation
  kubectl apply -Rf kubernetes/
  popd

  # Make wrk2 executable
  pushd DeathStarBench/wrk2
  make -j 4
  popd

  # Wait for the pods to get running
  sleep 1m

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
  pushd scripts
  kubectl apply -f deployment/hotel-reservation/reservation-ingress.yaml
  popd
  
  # Wait for the ingress to get running
  sleep 1m
fi

INGRESS_HOST=$(kubectl -n ingress-nginx get service ingress-nginx-controller -o jsonpath='{.spec.clusterIP}')
INGRESS_PORT=$(kubectl -n ingress-nginx get svc ingress-nginx-controller -o jsonpath='{.spec.ports[?(@.name=="http")].port}')

GATEWAY_URL=$INGRESS_HOST:$INGRESS_PORT

# Warm-up
pushd DeathStarBench/hotelReservation
../wrk2/wrk -D exp -t 5 -c 10 -d 10 -L -s ./wrk2/scripts/hotel-reservation/mixed-workload_type_1.lua http://$GATEWAY_URL -R 100

# Measure CPU and Memory
sudo python3 $TESTBED/scripts/utils/cpumem_stats.py reservation_linkerd > $TESTBED/logs/python.log 2>&1 &

# Run queries to log timings
../wrk2/wrk -D exp -t 10 -c 10 -d 40 -L -s ./wrk2/scripts/hotel-reservation/mixed-workload_type_1.lua http://$GATEWAY_URL -R 50 >> $TESTBED/out/time_reservation_linkerd.run 2>&1

# Kill CPU and Memory measurement
pid=$(pgrep -f cpumem)
sudo kill -SIGINT $pid

popd