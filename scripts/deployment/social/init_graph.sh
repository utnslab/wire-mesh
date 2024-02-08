# Initialize the social network graph
# Args:
#  $1: IP address of the social graph service
#  $2: Port of the social graph service

: "${TESTBED:=$HOME}"

pushd $TESTBED/DeathStarBench/socialNetwork
python scripts/init_social_graph.py --ip $1 --port $2
popd