"""
Get a summary of CPU and memory usage from the pickle generated by cpumem_stats.py
"""

import os
import sys
import pickle
import numpy as np

if len(sys.argv) < 2:
    print('Usage: python cpumem_summary.py <mesh_type> <file location>')
    sys.exit(1)

MESH = sys.argv[1]
DIR = sys.argv[2]

mean_mem = 0
mean_cpu = 0

for i in range(4):
    PKL_FILE = os.path.join(DIR, 'stats_{0}_{1}.pkl'.format(MESH, i))

    with open(PKL_FILE, "rb") as f:
        mem_usage = pickle.load(f)

    used_mem = mem_usage['used']
    cpu_usage = mem_usage['cpu']

    mean_mem += np.mean(used_mem)
    mean_cpu += np.mean(cpu_usage)

print('Average memory usage: {0} MB'.format(mean_mem))
print('Average CPU usage: {0}%'.format(mean_cpu))