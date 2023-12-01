#!/usr/bin/env bash
# Run query for the BookInfo benchmark with various service meshes
# Arguments:
# --mesh: service mesh name
# --init: whether first time install
# --server: whether to start the stats server
# --client: whether to start the stats client
# --ip: IP address of the stats server
# --port: port of the stats server

showHelp() {
cat << EOF  
Usage: <script_name> -m <mesh-name> [-isc] [-I <ip>]
Attach bpf programs for a specific service.

-h, -help,      --help        Display help
-m, -mesh,      --mesh        Service mesh name to put in the output file
-i, -init,      --init        Whether first time install
-s, -server,    --server      Whether to start the stats server
-c, -client,    --client      Whether to start the stats client
-I, -ip,        --ip          IP address of the stats server
-P, -port,      --port        Port of the stats server

EOF
}

MESH=""
INIT=0
SERVER=0
CLIENT=0
IP=""
PORT="32000"

options=$(getopt -l "help,mesh:,init,server,client,ip:,port:" -o "hm:iscI:P:" -a -- "$@")

eval set -- "$options"

while true; do
  case "$1" in
  -h|--help) 
      showHelp
      exit 0
      ;;
  -m|--mesh)
      shift
      MESH=$1
      ;;
  -i|--init)
      INIT=1
      ;;
  -c|--client)
      CLIENT=1
      ;;
  -s|--server)
      SERVER=1
      ;;
  -I|--ip)
      shift
      IP=$1
      ;;
  -P|--port)
      shift
      PORT=$1
      ;;
  --)
      shift
      break;;
  esac
  shift
done

: "${TESTBED:=$HOME}"

if [[ $INIT -eq 1 ]]; then
  # Init
  pushd $TESTBED/scripts
  if [[ $MESH == "wire" ]]; then
    kubectl apply -f deployment/bookinfo/wire-manifests.yaml
  elif [[ $MESH == "wire-partial" ]]; then
    ./deployment/bookinfo/wire_init.sh
  else
    kubectl apply -f deployment/bookinfo/bookinfo.yaml
  fi

  # Check if the mesh is istio
  if [[ $MESH == "istio" ]]; then
    # Use the Istio Ingress Gateway
    kubectl apply -f deployment/bookinfo/bookinfo-gateway.yaml
  else
    # Use Kubernetes Ingress NGINX
    kubectl apply -f deployment/bookinfo/bookinfo-ingress.yaml
  fi
  popd

  # Wait for the pods to get running
  sleep 1m
fi

if [[ $SERVER -eq 1 ]]; then
  # Start the stats server
  echo "Starting the stats server with IP=$IP"
  if [[ $MESH == "istio" ]]; then
    INGRESS_PORT=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.spec.ports[?(@.name=="http2")].nodePort}')
    echo "Ingress port for Istio is $INGRESS_PORT"
  fi

  sudo python3 $TESTBED/scripts/utils/stats_server.py $IP > $TESTBED/logs/python.log 2>&1 &
  wait
fi

if [[ $CLIENT -eq 1 ]]; then
  GATEWAY_URL="$IP:$PORT"
  if [[ $SERVER -eq 1 ]]; then
    if [[ $MESH == "istio" ]]; then
      INGRESS_HOST=$(kubectl get po -l istio=ingressgateway -n istio-system -o jsonpath='{.items[0].status.hostIP}')
      INGRESS_PORT=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.spec.ports[?(@.name=="http2")].nodePort}')
    else
      INGRESS_HOST=$(kubectl -n ingress-nginx get service ingress-nginx-controller -o jsonpath='{.spec.clusterIP}')
      INGRESS_PORT=$(kubectl -n ingress-nginx get svc ingress-nginx-controller -o jsonpath='{.spec.ports[?(@.name=="http")].port}')
    fi

    GATEWAY_URL=$INGRESS_HOST:$INGRESS_PORT
  fi

  echo "Starting the test with GATEWAY_URL=$GATEWAY_URL"

  # Run queries to log timings
  TIMEFORMAT=%R

  # Warm-up
  for i in $(seq 1 20); do
  { time curl -s -o /dev/null "http://$GATEWAY_URL/productpage"; } 2>&1
  done
  
  # Start the stats client
  sudo python3 $TESTBED/scripts/utils/stats_client.py $TESTBED/scripts/utils/config bookinfo_$MESH > $TESTBED/logs/python.log 2>&1 &
  
  # Run queries to log timings
  pushd $TESTBED/DeathStarBench/wrk2
  ../wrk2/wrk -D exp -t 10 -c 10 -d 60 -L http://$GATEWAY_URL/productpage -R 500 >> $TESTBED/out/time_bookinfo_$MESH.run 2>&1
  popd

  # Kill CPU and Memory measurement
  pid=$(pgrep -f stats_client)
  sudo kill -SIGINT $pid
fi