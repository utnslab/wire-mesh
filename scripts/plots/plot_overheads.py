"""
Python script to plot the overheads of using Istio proxy.
"""

import os
import sys
import numpy as np
import pickle as pkl
from plot_wrk_comp import get_latencies
import matplotlib.pyplot as plt

if len(sys.argv) < 2:
    print("Usage: python3 plot_overheads.py <path>")
    sys.exit(1)

PATH = sys.argv[1]

IS_MOTIVATION = "motivation" in PATH

if IS_MOTIVATION:
    LABELS = {
        'none': 'Plain',
        '1': 'I Tier',
        '2': 'II Tier',
        '3': 'III Tier',
        'all': 'All Tiers',
    }
else:
    LABELS = {
        'none': 'Plain',
        'ebpf': 'eBPF',
        'all': 'All Tiers',
    }

PERC = ['50', '99']
NODES = 4

# Plot formatting
plt.rcParams['text.usetex'] = True  #Let TeX do the typsetting
plt.rcParams['font.size'] = 14
# plt.rcParams['text.latex.preamble'] = [
#     r'\usepackage{sansmath}', r'\sansmath'
# ]  #Force sans-serif math mode (for axes labels)
plt.rcParams['font.family'] = 'sans-serif'  # ... for regular text
plt.rcParams[
    'font.sans-serif'] = 'Computer Modern Sans serif'  # Choose a nice font here

# Plot the graph
if IS_MOTIVATION:
    fig, (ax1, ax2, ax3) = plt.subplots(1, 3, figsize=(10, 2.6), gridspec_kw={'width_ratios': [3, 2, 2]})
else:
    fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(5.9, 3), gridspec_kw={'width_ratios': [4, 3]})

colors = ['#82B366', '#D79B00', '#B85450', '#6C8EBF', '#9673A6', '#D6B656']
markers = ['o', 'P', '^', 's', 'v', '^']
styles = ['-', '--', '-.', 'dotted']

labels_list = list(LABELS.keys())

def get_stats(type):
    stats = {}
    for dir, _, _ in os.walk(PATH):
        dirname = dir.split('/')[-1]
        if '-' not in dirname:
            continue

        app = dirname.split('-')[0]
        label = dirname.split('-')[1]

        if label in LABELS:
            all_stats = None
            for n in range(NODES):
                with open(
                        os.path.join(dir, 'stats_{0}_{1}_{2}.pkl'.format(app, label, n)),
                        'rb') as f:
                    stats_node = pkl.load(f)
                    if all_stats is None:
                        all_stats = stats_node[type]
                    else:
                        all_stats = np.sum(
                            [all_stats, stats_node[type]], axis=0)
            stats[label] = all_stats

    # Get the minimum length of the stats
    min_len = 100000
    for _, s in stats.items():
        min_len = min(min_len, len(s))

    offset = min_len - 60
    if min_len > 60:
        stats = {
            k: v[5+offset:min_len] for k, v in stats.items()
        }
    else:
        stats = {k: v[5:min_len] for k, v in stats.items()}

    # Print the analysis
    # print(appl, type)
    for l in LABELS:
        # Print how many times the mean is higher compared to plain
        if l != 'none':
            print(l, 1 - np.mean(stats[l]) / np.mean(stats['none']))

    return stats


# Latency overheads plot.
# Construct the latencies object.
latencies = {}
for p in PERC:
    latencies[p] = [None] * len(LABELS)

# Walk through the directory and get the latencies.
for dir, _, _ in os.walk(PATH):
    dirname = dir.split('/')[-1]
    if '-' not in dirname:
        continue

    app = dirname.split('-')[0]
    label = dirname.split('-')[1]
    rate = int(dirname.split('-')[2])

    if label in LABELS:
        # Get the percentile data
        file = os.path.join(dir, 'time_{0}_{1}_{2}.run'.format(app, rate, label))
        
        # Check if the file exists
        if not os.path.exists(file):
            continue
        
        with open(file) as f:
            content = f.readlines()
        content = [x.strip().split() for x in content]
        latencies_dict = get_latencies(content)

        for p in PERC:
            latencies[p][labels_list.index(label)] = latencies_dict[p]

print(latencies)

# Plot the latencies - two line curves, one for each percentile
ax1.plot(labels_list, latencies['50'],
         label='50 \%ile', color=colors[0],
         marker=markers[0], linestyle=styles[0], linewidth=2, markersize=15)
ax1.plot(labels_list, latencies['99'],
         label='99 \%ile', color=colors[1],
         marker=markers[1], linestyle=styles[1], linewidth=2, markersize=15)

# Set the font of y-axis to be larger
ax1.set_yticks(np.arange(0, 35, 10))
ax1.tick_params(axis='both', which='major', labelsize=22)
ax1.set_ylabel('Latency (ms)', fontsize=24)
ax1.legend(fontsize=18, loc='upper left')


# CPU overheads plot.
cpu_stats = get_stats('cpu')

cpu_plot = []
labels_plot = []
for l in LABELS:
    # if l != 'none':
    #     cpu_plot.append(np.mean(cpu_stats[l]) - np.mean(cpu_stats['none']))
    #     labels_plot.append(l)
    cpu_plot.append(np.mean(cpu_stats[l]))
    labels_plot.append(l)
print(cpu_plot)

# Plot the CPU usage as a bar chart
ax2.bar(labels_plot, cpu_plot, color=colors[2])
ax2.set_yticks(np.arange(0, 14, 4))
ax2.tick_params(axis='both', which='major', labelsize=22)
ax2.set_ylabel('CPU Usage (\%)', fontsize=24)

if IS_MOTIVATION:
    # Memory overheads plot.
    mem_stats = get_stats('used')

    mem_plot = []
    labels_plot = []
    for l in LABELS:
        # if l != 'none':
        #     mem_plot.append(np.mean(mem_stats[l]) - np.mean(mem_stats['none']))
        #     labels_plot.append(l)
        mem_plot.append(np.mean(mem_stats[l]) / 1000)
        labels_plot.append(l)
    print(mem_plot)

    ax3.bar(labels_plot, mem_plot, color=colors[3])
    ax3.set_ylim([0, 12])
    ax3.set_yticks(np.arange(0, 14, 4))
    ax3.tick_params(axis='both', which='major', labelsize=22)
    ax3.set_ylabel('Mem. Usage (GB)', fontsize=24)


# ax2.set_xlabel('Tiers of the microservice graph where the proxy is injected', fontsize=26)

fig.tight_layout()
plt.subplots_adjust(left=0.08, right=0.98, top=1, bottom=0.2, wspace=0.5)
# fig.suptitle('Tiers of microservice graph where proxy was injected', fontsize=26, y=0)

if IS_MOTIVATION:
    plt.savefig('overheads.png', bbox_inches='tight')
    plt.savefig('overheads.pdf', bbox_inches='tight')
else:
    plt.savefig('ebpf-overheads.pdf', bbox_inches='tight')
    plt.savefig('ebpf-overheads.png', bbox_inches='tight')