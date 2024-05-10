"""
Python script to plot the overheads of using Istio proxy.
"""

import os
import sys

import matplotlib.pyplot as plt

if len(sys.argv) < 2:
    print("Usage: python3 plot_overheads.py <path>")
    sys.exit(1)

PATH = sys.argv[1]

# Read "overheads.csv".
overheads = {}
with open(os.path.join(PATH, 'overheads.csv'), 'r') as f:
    for line in f:
        parts = line.strip().split(',')
        conns = parts[1]
        qps = parts[2]
        latency = {
            'p50': parts[3],
            'p95': parts[4],
            'p99': parts[5],
            'p99.9': parts[6]
        }