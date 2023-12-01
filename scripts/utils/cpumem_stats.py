"""Start measuring and trap INTERRUPT signal to gracefully end execution"""
import os
import sys
import time
import datetime
import pickle
import psutil
import signal

# End gracefully on interrupt
def signal_handler(sig, frame):
    # Write the stats to a pickle file, which can be read by the plot macro script
    mem_usage = {
        'used': used_mem,
        'cpu': cpu_usage
    }
    with open(PKL_FILE, "wb") as f:
        pickle.dump(mem_usage, f)

    sys.exit(0)

if len(sys.argv) < 2:
    print('Usage: python cpumem_stats.py <mesh_type>')
    sys.exit(1)

# Read the mesh.txt file to get the mesh type
MESH = sys.argv[1]

# Read the $TESTBED environment variable
testbed = os.environ.get('TESTBED')
if testbed is None:
    # Set testbed to be $HOME
    testbed = os.environ.get('HOME')

PKL_FILE = os.path.join(testbed + '/out', 'stats_{0}.pkl'.format(MESH))

# Register a signal handler
signal.signal(signal.SIGINT, signal_handler)

# Hold the memory stats
total_memory = psutil.virtual_memory().total / (1024**2)  # In MB
used_mem = []
cpu_usage = []

last_update_time = datetime.datetime.now()

# Continuously monitor usage every second
while True:
    avbl_mem = psutil.virtual_memory().available / (1024**2)
    used_mem.append(total_memory - avbl_mem)
    print(datetime.datetime.now(), avbl_mem)

    cpu = psutil.cpu_percent()
    cpu_usage.append(cpu)

    mem_usage = {
        'used': used_mem,
        'cpu': cpu_usage
    }
    with open(PKL_FILE, "wb") as f:
        pickle.dump(mem_usage, f)

    diff = datetime.datetime.now() - last_update_time
    elapsed_time = int((diff.seconds * 1000) + (diff.microseconds / 1000))
    time.sleep((1000 - elapsed_time) / 1000)
    last_update_time = datetime.datetime.now()
