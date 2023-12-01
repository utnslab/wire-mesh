"""
Plot the comparison of TCP Sockets vs Socket Redirection vs Shared Memory
Arguments:
1: The path to the folder containing the csv file.
"""

import os
import sys
import numpy as np
import matplotlib.pyplot as plt

if len(sys.argv) < 2:
    print("Usage: python3 plot_redirection.py <path>")
    sys.exit(1)

path = sys.argv[1]

# Read the csv file
with open(os.path.join(path, 'proxy-acceleration.csv') ) as f:
    content = f.readlines()
content = [x.strip().split(',') for x in content[1:]]

# Get the data
x = [int(x[0]) for x in content]
y_tcp = [float(x[1]) for x in content]
y_redir = [float(x[2]) for x in content]
y_shm = [float(x[3]) for x in content]

# Plot formatting
plt.rcParams['text.usetex'] == True
plt.rcParams['pdf.fonttype'] = 42
plt.rcParams['font.family'] = ['sans-serif']  # ... for regular text
plt.rcParams['font.sans-serif'] = ['Helvetica'] + plt.rcParams['font.sans-serif'] # Choose a nice font here

# Bar plot to compare the various percentile of latencies
fig = plt.figure(figsize=(10, 4))  # 6.4:4.8
ax = fig.add_subplot(111)

# Plot the data
xticks = np.linspace(0, 10000, 11)
yticks = np.linspace(0, 16, 9)
ax.plot(x, y_tcp, '-', label='TCP Sockets', linewidth=2)
ax.plot(x, y_redir, '-.', label='Socket Redirection', linewidth=2)
# ax.plot(x, y_shm, '--', label='Shared Memory', linewidth=2)
ax.set_xticks(xticks)
ax.set_xticklabels([int(x) for x in xticks], fontsize=16)
ax.set_xlabel('Data Size (in B)', fontsize=20, fontweight='bold')
ax.set_yticks(yticks)
ax.set_yticklabels(yticks, fontsize=16)
ax.set_ylabel('Latency (in us)', fontsize=20, fontweight='bold')
ax.legend(fontsize=16, ncol=3, loc='upper center')
plt.subplots_adjust(left=0.1, right=0.95, top=0.95, bottom=0.15)
plt.savefig(os.path.join(path, 'proxy-acceleration.png'))
plt.savefig(os.path.join(path, 'proxy-acceleration.pdf'))