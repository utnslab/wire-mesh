#!/bin/bash
# Detach sockops and fast path bpf program for a specific service.

showHelp() {
cat << EOF  
Usage: <script_name> -s <service-name> [-okmc]
Attach bpf programs for a specific service.

-h, -help,      --help        Display help
-s, -service,   --service     Service Name
-o, -ops,       --ops         Detach sockops and fast path if set, else fast path only
-k, -skb,       --skb         Detach SKB program
-m, -skmsg,     --skmsg       Detach SK_MSG program
-c, --control,  --control     Whether running the script on the control node

EOF
}

SERVICE=""
OPS=0
SKB=0
SKMSG=0
CONTROL=0

options=$(getopt -l "help,service:,ops,skb,skmsg,control" -o "hs:okmc" -a -- "$@")

eval set -- "$options"

while true; do
  case "$1" in
  -h|--help) 
      showHelp
      exit 0
      ;;
  -s|--service)
      shift
      SERVICE=$1
      ;;
  -o|--ops)
      OPS=1
      ;;
  -k|--skb)
      SKB=1
      ;;
  -m|--skmsg)
      SKMSG=1
      ;;
  -c|--control)
      CONTROL=1
      ;;
  --)
      shift
      break;;
  esac
  shift
done

: "${TESTBED:=$HOME}"
pushd $TESTBED/scripts

KUBECTL="kubectl"

# If not on control node, then change the kubectl command.
if [ $CONTROL -eq 0 ]; then
  KUBECTL="kubectl --kubeconfig $HOME/admin.conf"
fi

POD_IPS=$($KUBECTL get endpoints $SERVICE -o=jsonpath='{.subsets[*].addresses[*].ip}')
CMD="$KUBECTL get pods -A -o custom-columns=PodName:.metadata.name,PodUID:.metadata.uid,ContainerID:.status.containerStatuses[0].containerID,PodIP:.status.podIP"

$CMD | grep $SERVICE | while read -r POD_INFO; do
  POD_IP=$(echo $POD_INFO | awk '{print $4}')
  if [[ ! $POD_IPS =~ $POD_IP ]]; then
    continue
  fi

  # Extract Pod UID and Container ID.
  POD_UID=$(echo $POD_INFO | awk '{print $2}')
  CONTAINER_ID=$(echo $POD_INFO | awk '{print $3}' | sed 's/containerd:\/\///')

  # Replace '-' with '_' in the Pod UID.
  POD_UID=$(echo $POD_UID | sed 's/-/_/g')

  # Get the cgroup path for the pod.
  POD_CGROUP="kubepods-burstable-pod${POD_UID}.slice:cri-containerd:${CONTAINER_ID}"

  # Check if the POD_CGROUP exists.
  BESTEFFORT=0
  if [ ! -d "/sys/fs/cgroup/unified/$POD_CGROUP" ]; then
    # Use the besteffort cgroup if burstable is not found.
    POD_CGROUP="kubepods-besteffort-pod${POD_UID}.slice:cri-containerd:${CONTAINER_ID}"
    BESTEFFORT=1
  fi

  # If neither burstable nor besteffort is found, then skip this pod.
  if [ ! -d "/sys/fs/cgroup/unified/$POD_CGROUP" ]; then
    continue
  fi

  # Detach sockops program from the pod's cgroup.
  pushd $TESTBED/bpf-pathprop/path-router
  if [ $SKB -eq 1 ]; then
    echo "Detaching SKB bpf program from $SERVICE ..."
    sudo ./load_sk_skb --cgroup $POD_CGROUP --unload
  fi

  if [ $SKMSG -eq 1 ]; then
    echo "Detaching gRPC parser bpf program from $1 ..."
    sudo ./load_grpc_skmsg --cgroup $POD_CGROUP --unload
  fi

  if [ $OPS -eq 1 ]; then
    echo "Detaching sockops bpf program from $1 ..."
    sudo ./load_sockops --cgroup $POD_CGROUP --unload
  fi
  popd

  # Remove the bpf maps.
  DIR_NAME="burstable"
  if [ $BESTEFFORT -eq 1 ]; then
    DIR_NAME="besteffort"
  fi

  if [ $OPS -eq 1 ]; then
    sudo rm -r /sys/fs/bpf/kubepods-${DIR_NAME}-pod${POD_UID}-sockops
  fi

  if [ $SKB -eq 1 ] || [ $SKMSG -eq 1 ]; then
    sudo rm -r /sys/fs/bpf/kubepods-${DIR_NAME}-pod${POD_UID}
  fi

  sudo rm -r /sys/fs/bpf/kubepods-${DIR_NAME}-pod${POD_UID}-prog

done

popd