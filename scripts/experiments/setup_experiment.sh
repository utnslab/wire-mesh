#!/bin/bash

showHelp() {
cat << EOF  
Usage: <script_name> [-a <appl>] [-m <mesh>] [-h]
Setup application and mesh **on the control node**.

-h, -help,      --help        Display help
-a, -app,       --app         Application to run (reservation, bookinfo, boutique, social-network)
-m, -mesh,      --mesh        Name of the mesh to use (istio, linkerd, plain, nginx)

EOF
}

MESH="istio"
APP="reservation"

options=$(getopt -l "help,mesh:,app:" -o "hm:a:" -a -- "$@")

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
  -a|--app)
      shift
      APP=$1
      ;;
  --)
      shift
      break;;
  esac
  shift
done

# Setup service mesh for the given argument
: "${TESTBED:=$HOME}"
pushd $TESTBED/scripts

# Install service mesh on the control node.
if [[ "$MESH" == "istio" ]] && [[ "$APPL" == "boutique" || "$APPL" == "bookinfo" ]]; then
  ./mesh_setup.sh --mesh $MESH --ingress
else
  ./mesh_setup.sh --mesh $MESH
fi

# Initialize application on the control node
./deployment/$APP/run_query.sh --init --mesh $MESH