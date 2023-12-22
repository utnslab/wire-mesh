#!/usr/bin/env bash
# Run query for the Hotel Reservation benchmark with different service mesh
# Arguments:
# --mesh: service mesh name
# --init: whether first time install
# --server: whether to start the stats server
# --client: whether to start the stats client
# --ip: IP address of the stats server

showHelp() {
cat << EOF  
Usage: <script_name> -m <mesh-name> [-isc] [-I <ip>] [-r <rate>]
Attach bpf programs for a specific service.

-h, -help,      --help        Display help
-m, -mesh,      --mesh        Service mesh name to put in the output file
-i, -init,      --init        Whether first time install
-s, -server,    --server      Whether to start the stats server
-c, -client,    --client      Whether to start the stats client
-I, -ip,        --ip          IP address of the stats server
-r, -rate,      --rate        Rate of requests per second

EOF
}

MESH=""
INIT=0
SERVER=0
CLIENT=0
IP=""
RATE=2000

options=$(getopt -l "help,mesh:,init,server,client,ip:" -o "hm:iscI:r:" -a -- "$@")

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
  -r|--rate)
      shift
      RATE=$1
      ;;
  --)
      shift
      break;;
  esac
  shift
done

: "${TESTBED:=$HOME}"
pushd $TESTBED

if [[ $INIT -eq 1 ]]; then
  sudo apt install -y luarocks
  sudo luarocks install luasocket

  # Pull docker image 
  docker pull divyanshus/hotelreservation
  
  if [ ! -d "$TESTBED/DeathStarBench" ]; then
    git clone https://github.com/DivyanshuSaxena/DeathStarBench.git

    # Make wrk2 executable
    pushd DeathStarBench/wrk2
    make -j 4
    popd
  fi

  if [[ $MESH == "wire" ]]; then
    pushd scripts/deployment/reservation
    ./wire_init.sh
    popd
  else
    pushd DeathStarBench/hotelReservation
    kubectl apply -Rf kubernetes/
    popd
  fi

  # Wait for the pods to get running
  sleep 1m
fi

if [[ $SERVER -eq 1 ]]; then
  # Start the stats server
  echo "Starting the stats server with IP=$IP"
  sudo python3 $TESTBED/scripts/utils/stats_server.py $IP > $TESTBED/logs/python.log 2>&1 &
  wait
fi

if [[ $CLIENT -eq 1 ]]; then
  GATEWAY_URL="$IP:32000"
  if [[ $SERVER -eq 1 ]]; then
    echo "Setting it to kubectl service endpoint"
    INGRESS_HOST=$(kubectl get svc frontend -o jsonpath='{.spec.clusterIP}')
    INGRESS_PORT=$(kubectl get svc frontend -o jsonpath='{.spec.ports[0].port}')

    GATEWAY_URL=$INGRESS_HOST:$INGRESS_PORT
  fi

  echo "Starting the test with GATEWAY_URL=$GATEWAY_URL"

  # Warm-up
  pushd DeathStarBench/hotelReservation
  ../wrk2/wrk -D exp -t 5 -c 5 -d 60 -L -s ./wrk2/scripts/hotel-reservation/mixed-workload_type_1.lua http://$GATEWAY_URL -R 10

  # Start the stats client
  sudo python3 $TESTBED/scripts/utils/stats_client.py $TESTBED/scripts/utils/config reservation_$MESH > $TESTBED/logs/python.log 2>&1 &

  sleep 30

  # Run queries to log timings
  ../wrk2/wrk -D exp -t 10 -c 10 -d 60 -L -s ./wrk2/scripts/hotel-reservation/mixed-workload_type_1.lua http://$GATEWAY_URL -R $RATE >> $TESTBED/out/time_reservation_${RATE}_${MESH}.run 2>&1

  # Kill CPU and Memory measurement
  pid=$(pgrep -f stats_client)
  sudo kill -SIGINT $pid
fi

popd
