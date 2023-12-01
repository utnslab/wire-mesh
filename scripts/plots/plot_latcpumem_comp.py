"""
Plot latency, cpu and memory usage for each of the different policy sets.
"""

import os
import sys
import json
import numpy as np
import pickle as pkl
import matplotlib.pyplot as plt
import matplotlib.patches as mpatches

from plot_wrk_comp import get_latencies
from matplotlib.ticker import ScalarFormatter

if len(sys.argv) < 2:
    print("Usage: python3 plot_latcpumem_comp.py <path>")
    sys.exit(1)

policies = ['single', 'double', 'all']
num_policies = len(policies)
num_nodes = 4

directory = sys.argv[1]

# Read eval_config.json to get directories for each application
current_dir = os.path.dirname(os.path.realpath(__file__))
json_file = os.path.join(current_dir, 'eval_config.json')

with open(json_file) as f:
    results_data = json.load(f)

# Plot percentiles
pct_to_plot = ['50', '90', '99', '99.9']

# Plot formatting
plt.rcParams['text.usetex'] = True  #Let TeX do the typsetting
plt.rcParams['font.size'] = 14
plt.rcParams['text.latex.preamble'] = [
    r'\usepackage{sansmath}', r'\sansmath'
]  #Force sans-serif math mode (for axes labels)
plt.rcParams['font.family'] = 'sans-serif'  # ... for regular text
plt.rcParams[
    'font.sans-serif'] = 'Computer Modern Sans serif'  # Choose a nice font here

a_val = 1.0
colors = ['#D5E8D4', '#FFE6CC', '#F8CECC']
edge_colors = ['#82B366', '#D79B00', '#B85450']
hatches = ['//', '..', '\\\\']
label1 = mpatches.Patch(facecolor=colors[0],
                        edgecolor=edge_colors[0],
                        alpha=a_val,
                        hatch='//',
                        label='Linkerd Heavy')
label2 = mpatches.Patch(facecolor=colors[1],
                        edgecolor=edge_colors[1],
                        alpha=a_val,
                        hatch='..',
                        label='Istio Heavy')
label3 = mpatches.Patch(facecolor=colors[2],
                        edgecolor=edge_colors[2],
                        alpha=a_val,
                        hatch=r'\\\\',
                        label='All Istio')


def get_stats(type):
    stats = []
    min_len = 10000
    adjustment_wire = 5600
    for p in range(num_policies):
        # Get the directory for the application
        appl_dir = results_data['reservation']['mixed'][p]

        stats_wire = None
        for n in range(num_nodes):
            if p == 0 or p == 1:
                with open(
                        os.path.join(directory, appl_dir,
                                    'stats_reservation_mixed_{0}.pkl'.format(n)),
                        'rb') as f:
                    stats_node = pkl.load(f)
                    if stats_wire is None:
                        stats_wire = stats_node[type]
                    else:
                        stats_wire = np.sum([stats_wire, stats_node[type]], axis=0)
            else:
                with open(
                        os.path.join(directory, appl_dir,
                                    'stats_reservation_istio_{0}.pkl'.format(n)),
                        'rb') as f:
                    stats_node = pkl.load(f)
                    if stats_wire is None:
                        stats_wire = stats_node[type]
                    else:
                        stats_wire = np.sum([stats_wire, stats_node[type]], axis=0)

        # Get the minimum length of the stats
        min_len = min(len(stats_wire), min_len)
        offset = min_len - 60

        if type == 'used':
            stats_wire -= adjustment_wire

        if min_len > 60:
            stats.append(stats_wire[5 + offset:min_len])
        else:
            stats.append(stats_wire[5:min_len])

    return stats



def add_boxplot(ax, stats_data, ylabel):
    # Three box plots for each application. Each box plot has 3 boxes for each service mesh.
    # Color each service mesh differently.
    for j in range(num_policies):
        # Plot the box plots
        ax.boxplot(
            stats_data[j],
            positions=[j],
            widths=0.4,
            showfliers=False,
            patch_artist=True,
            boxprops=dict(facecolor=colors[j],
                          color=edge_colors[j],
                          hatch=hatches[j]),
            capprops=dict(color=edge_colors[j]),
            whiskerprops=dict(color=edge_colors[j]),
            flierprops=dict(color=edge_colors[j],
                            markeredgecolor=edge_colors[j]),
            medianprops=dict(color=edge_colors[j], linewidth=2))

        ax.set_xlabel(ylabel, fontsize=22)
        # if ylabel == 'Memory Usage (MB)':
        #     ax.set_yticks([4800, 5000, 5200, 5400, 5600, 5800])
        ax.set_yticklabels([int(y) for y in ax.get_yticks()], fontsize=20)
        ax.tick_params(axis='x', which='both', bottom=False, top=False, labelbottom=False)

        # Set limit [10, 40] for cpu usage and [10500, 11500] for memory usage
        if ylabel == 'CPU Usage (\%)':
            ax.set_ylim([20, 40])


latencies = []
for p in range(num_policies):
    # Get the directory for the application
    appl_dir = results_data['reservation']['mixed'][p]

    # Get the percentile data
    if p == 0 or p == 1:
        latency_file = os.path.join(directory, appl_dir,
                                    'time_reservation_mixed.run')
    else:
        latency_file = os.path.join(directory, appl_dir,
                                    'time_reservation_istio.run')

    with open(latency_file) as f:
        content = f.readlines()
    content = [x.strip().split() for x in content]

    latencies.append(get_latencies(content))

cpu_stats = get_stats('cpu')
mem_stats = get_stats('used')

xs = np.arange(4)
width = 0.25

# Make three subplots, one for latency, one for cpu and one for memory.
fig = plt.figure(figsize=(12, 3.6))

ax0 = fig.add_subplot(1, 4, (1, 2))
ax1 = fig.add_subplot(1, 4, (3, 3))
ax2 = fig.add_subplot(1, 4, (4, 4))

# Use ax0 for latency
ax0.bar(xs - width, [latencies[0][p] for p in pct_to_plot],
            width=width,
            edgecolor=edge_colors[0],
            color=colors[0],
            hatch='//')
ax0.bar(xs, [latencies[1][p] for p in pct_to_plot],
            width=width,
            edgecolor=edge_colors[1],
            color=colors[1],
            hatch='..')
ax0.bar(xs + width, [latencies[2][p] for p in pct_to_plot],
            width=width,
            edgecolor=edge_colors[2],
            color=colors[2],
            hatch='\\\\')

ax0.set_xticks(xs)
labels = [p + 'th' for p in pct_to_plot]
ax0.set_xticklabels(labels, fontsize=20)
ax0.set_xlabel('Percentile', fontsize=22)

ys = [int(y) for y in np.linspace(0, 60000, 7)]
ax0.set_yticks(ys)
ax0.set_yticklabels([str(y) for y in ys])
ax0.tick_params(axis='y', labelsize=20)
ax0.set_ylabel('Latency (ms)', fontsize=22)

# Use ax1 for cpu
add_boxplot(ax1, cpu_stats, 'CPU Usage\n(\%)')

# Use ax2 for memory
add_boxplot(ax2, mem_stats, 'Memory Usage\n(MB)')

fig.legend(handles=[label1, label2, label3],
           loc='center',
           ncol=3,
           frameon=False,
           bbox_to_anchor=(0.5, 0.93),
           bbox_transform=fig.transFigure,
           fontsize=20)

fig.tight_layout()
plt.subplots_adjust(left=0.1, right=0.98, top=0.87, bottom=0.2)
plt.savefig('mixed-comp.pdf')
plt.savefig('mixed-comp.png')
