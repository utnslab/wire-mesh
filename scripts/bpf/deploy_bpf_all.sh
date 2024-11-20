# Attach BPF add-ons by running this script on the control node.

: "${TESTBED:=$HOME}"
pushd $TESTBED

# First run on the control node
pushd $TESTBED/scripts/bpf
./all_services.sh --control
popd

# SSH and run on every node except the control node
# Get hostnames from kubectl get nodes
HOSTS=$(kubectl get nodes | grep node | awk '{print $1}')

for host in $HOSTS; do
  # Skip the control node
  if [[ $host == *"node0"* ]]; then
    continue
  fi

  ssh -o StrictHostKeyChecking=no $host "cd \$HOME/scripts/bpf; ./all_services.sh"
done