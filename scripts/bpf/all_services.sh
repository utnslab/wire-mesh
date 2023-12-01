#!/bin/bash
# Attach or Detach all sockops and fast path bpf programs.
# Arguments:
# --detach: detach all sockops and fast path bpf programs
# --control: Whether running the script on the control node

showHelp() {
cat << EOF  
Usage: <script_name> -s <service-name> [-dc]
Attach bpf programs for a specific service.

-h, -help,      --help        Display help
-d, -detach,    --detach      Detach all sockops and fast path bpf programs
-c, --control,  --control     Whether running the script on the control node

EOF
}

DETACH=0
CONTROL=0

options=$(getopt -l "help,detach,control" -o "hdc" -a -- "$@")

eval set -- "$options"

while true; do
  case "$1" in
  -h|--help) 
      showHelp
      exit 0
      ;;
  -d|--detach)
      DETACH=1
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

SERVICES=$($KUBECTL get svc | awk '{print $1}' | tail -n +2)

pushd bpf
for svc in $SERVICES; do
    # If svc has consul or jaeger, then skip.
    if [[ $svc == *"consul"* ]] || [[ $svc == *"jaeger"* ]]; then
        continue
    fi

    if [ $DETACH -eq 1 ]; then
        if [ $CONTROL -eq 1 ]; then
            ./detach_bpf_service.sh --service $svc --ops --skmsg --skb --control
        else
            ./detach_bpf_service.sh --service $svc --ops --skmsg --skb
        fi
    else
        # Call the attach_bpf_service.sh script.
        if [ $CONTROL -eq 1 ]; then
            ./attach_bpf_service.sh --service $svc --ops --skmsg --skb --control
        else
            ./attach_bpf_service.sh --service $svc --ops --skmsg --skb
        fi
    fi
done
popd

popd