"""
Get comparison of CPU and Memory usage to compare the performance of plain, nginx and istio
Arguments:
1: the path to the folder containing the pickle files.
"""

import os
import sys
import pickle
import numpy as np
import matplotlib.pyplot as plt

if len(sys.argv) < 2:
    print("Usage: python3 plot_cpumem.py <path> <low/high>")
    sys.exit(1)

path = sys.argv[1]
LOWLOAD = sys.argv[2] == 'low'

applications = ['bookinfo', 'boutique', 'reservation']

for appl in applications:
    print(appl)
    dir_name = appl
    if LOWLOAD:
        dir_name += '-lowload'
    with open(os.path.join(path, '{0}/stats_{1}_plain.pkl'.format(dir_name, appl)), 'rb') as f:
        stats_plain = pickle.load(f)

    with open(os.path.join(path, '{0}/stats_{1}_nginx.pkl'.format(dir_name, appl)), 'rb') as f:
        stats_nginx = pickle.load(f)

    with open(os.path.join(path, '{0}/stats_{1}_istio.pkl'.format(dir_name, appl)), 'rb') as f:
        stats_istio = pickle.load(f)

    # Get the minimum length of the stats
    min_len = min(len(stats_plain['cpu']), len(stats_nginx['cpu']), len(stats_istio['cpu']))

    # Compare the mean CPU usage of the 3 cases
    cpu_plain = np.mean(stats_plain['cpu'])
    cpu_nginx = np.mean(stats_nginx['cpu'])
    cpu_istio = np.mean(stats_istio['cpu'])

    # Print the ratio of CPU usage
    print("Nginx: {:.2f}".format(cpu_nginx / cpu_plain))
    print("Istio: {:.2f}".format(cpu_istio / cpu_plain))

    plain_mem = np.average(stats_plain['used'])
    nginx_mem = np.average(stats_nginx['used'])
    istio_mem = np.average(stats_istio['used'])

    # Print the ratio of memory usage
    print("Nginx Mem: {:.2f}".format(nginx_mem / plain_mem))
    print("Istio Mem: {:.2f}".format(istio_mem / plain_mem))
