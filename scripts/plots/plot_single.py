"""
Plot the e2e latencies of requests sent to an application on various mesh types
Arguments:
1: Path to the directory containing the time_*.run files
2: Application
"""

import os
import sys
import matplotlib.pyplot as plt

path = sys.argv[1]
appl = sys.argv[2]

mesh = ['istio', 'linkerd', 'plain']

for mesh_type in mesh:
    with open(os.path.join(path, 'time_' + appl + '_' + mesh_type + '.run')) as f:
        content = f.readlines()
    content = [int(float(x.strip()) * 1000) for x in content]
    plt.plot(content, label=mesh_type)

plt.xlabel('Requests')
plt.ylabel('Latency (ms)')
plt.legend()
plt.savefig(os.path.join(path, appl + '_latencies.png'))
