"""
Make box plots for CPU and memory usage for each application and service mesh.
"""

import os
import sys
import json
import numpy as np
import pickle as pkl
import matplotlib.pyplot as plt
import matplotlib.patches as mpatches

if len(sys.argv) < 2:
    print("Usage: python3 plot_cpumem_comp.py <path> <low/high>")
    sys.exit(1)

num_applications = 3
num_nodes = 4
applications = ['bookinfo', 'boutique', 'reservation']
directory = sys.argv[1]
LOWLOAD = sys.argv[2] == 'low'

# Read eval_config.json to get directories for each application
current_dir = os.path.dirname(os.path.realpath(__file__))
json_file = os.path.join(current_dir, 'eval_config.json')

with open(json_file) as f:
    results_data = json.load(f)

# Plot formatting
plt.rcParams['text.usetex'] = True  #Let TeX do the typsetting
plt.rcParams['font.size'] = 14
plt.rcParams['text.latex.preamble'] = [
    r'\usepackage{sansmath}', r'\sansmath'
]  #Force sans-serif math mode (for axes labels)
plt.rcParams['font.family'] = 'sans-serif'  # ... for regular text
plt.rcParams[
    'font.sans-serif'] = 'Computer Modern Sans serif'  # Choose a nice font here

colors = ['#D5E8D4', '#FFE6CC', '#F8CECC']
edge_colors = ['#82B366', '#D79B00', '#B85450']
hatches = ['//', '..', '\\\\']

label1 = mpatches.Patch(facecolor=colors[0],
                        edgecolor=edge_colors[0],
                        hatch=hatches[0],
                        label='Wire - Best Case')
label2 = mpatches.Patch(facecolor=colors[1],
                        edgecolor=edge_colors[1],
                        hatch=hatches[1],
                        label='Wire - Worst Case')
label3 = mpatches.Patch(facecolor=colors[2],
                        edgecolor=edge_colors[2],
                        hatch=hatches[2],
                        label='Istio')


def get_stats(type):
    stats = []
    if LOWLOAD:
        adjustment_wire = 6200
        adjustment_istio = 6000
    else:
        adjustment_wire = 5600
        adjustment_istio = 5400

    for appl in applications:
        # Get the directory for the application
        appl_dir = results_data[appl]['heavy']
        if LOWLOAD:
            appl_dir = results_data[appl]['light']

        stats_wire_best = None
        stats_wire = None
        stats_istio = None
        for n in range(num_nodes):
            with open(
                    os.path.join(directory, appl_dir['wire-best'],
                                 'stats_{0}_wire_{1}.pkl'.format(appl, n)),
                    'rb') as f:
                stats_node = pkl.load(f)
                if stats_wire_best is None:
                    stats_wire_best = stats_node[type]
                else:
                    stats_wire_best = np.sum(
                        [stats_wire_best, stats_node[type]], axis=0)

            with open(
                    os.path.join(directory, appl_dir['wire'],
                                 'stats_{0}_wire_{1}.pkl'.format(appl, n)),
                    'rb') as f:
                stats_node = pkl.load(f)
                if stats_wire is None:
                    stats_wire = stats_node[type]
                else:
                    stats_wire = np.sum([stats_wire, stats_node[type]], axis=0)

            with open(
                    os.path.join(directory, appl_dir['istio'],
                                 'stats_{0}_istio_{1}.pkl'.format(appl, n)),
                    'rb') as f:
                stats_node = pkl.load(f)
                if stats_istio is None:
                    stats_istio = stats_node[type]
                else:
                    stats_istio = np.sum([stats_istio, stats_node[type]],
                                         axis=0)

        # Get the minimum length of the stats
        min_len = min(len(stats_wire), len(stats_istio))
        offset = min_len - 60

        # Adjust for extra memory usage
        if type == 'used':
            stats_wire_best -= adjustment_wire
            stats_wire -= adjustment_wire
            stats_istio -= adjustment_istio

        if min_len > 60:
            stats.append([
                stats_wire_best[5 + offset:min_len],
                stats_wire[5 + offset:min_len], stats_istio[5 + offset:min_len]
            ])
        else:
            stats.append([
                stats_wire_best[5:min_len], stats_wire[5:min_len],
                stats_istio[5:min_len]
            ])

        wire_best_mean = np.mean(stats[-1][0])
        wire_mean = np.mean(stats[-1][1])
        istio_mean = np.mean(stats[-1][2])

        print(appl, type)
        print('Istio/Wire Best: {0:.2f}'.format(istio_mean / wire_best_mean))
        print('Istio/Wire: {0:.2f}'.format(istio_mean / wire_mean))

    return stats


def add_boxplot(ax, stats_data, ylabel):
    # Three box plots for each application. Each box plot has 3 boxes for each service mesh.
    # Color each service mesh differently.
    for j in range(3):
        stats = []
        # Get the CPU usage for each application
        for i, appl in enumerate(applications):
            stats.append(stats_data[i][j])

        # Plot the box plots
        ax.boxplot(
            stats,
            positions=[i * 2 + j * 0.5 + 1 for i in range(len(applications))],
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

        # Set the xticks
        ax.set_xticks([i * 2 + 1.5 for i in range(len(applications))])
        ax.set_xticklabels(applications, fontsize=22)

        if ylabel == 'Memory Usage (MB)':
            if LOWLOAD:
                ax.set_yticks([4200, 4500, 4800, 5100, 5400, 5700])
            else:
                ax.set_yticks([4800, 5000, 5200, 5400, 5600, 5800])
        ax.set_yticklabels([int(y) for y in ax.get_yticks()], fontsize=20)

        ax.set_ylabel(ylabel, fontsize=22)
        ax.set_xlabel('Applications', fontsize=22)

        # Set limit [10, 40] for cpu usage and [10500, 11500] for memory usage
        if ylabel == 'CPU Usage (\%)':
            ax.set_ylim([0, 40])


# Make a boxplot for CPU usage
fig = plt.figure(figsize=(10.2, 3.6))

# Plot CPU usage
ax = fig.add_subplot(121)
cpu_stats = get_stats('cpu')
add_boxplot(ax, cpu_stats, 'CPU Usage (\%)')

# Plot memory usage
ax = fig.add_subplot(122)
mem_stats = get_stats('used')
add_boxplot(ax, mem_stats, 'Memory (MB)')

fig.legend(handles=[label1, label2, label3],
           loc='center',
           ncol=3,
           frameon=False,
           bbox_to_anchor=(0.5, 0.93),
           bbox_transform=fig.transFigure,
           fontsize=20)

# Save the plot
plt.tight_layout()
plt.subplots_adjust(top=0.87, bottom=0.2)
if not LOWLOAD:
    plt.savefig('memcpu-heavy.pdf')
    plt.savefig('memcpu-heavy.png')
else:
    plt.savefig('memcpu-light.pdf')
    plt.savefig('memcpu-light.png')
