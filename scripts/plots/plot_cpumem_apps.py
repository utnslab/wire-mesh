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
    print("Usage: python3 plot_cpumem_apps.py <path> <list of apps>")
    sys.exit(1)

NODES = 4
PATH = sys.argv[1]
APPS = sys.argv[2:]

UID = PATH.split('/')[-1]
LABELS = {
    'istio': 'Istio',
    'hypo': 'Istio++',
    'devbest': 'Multiple',
    'wire': 'Wire'
}

MESH = ["istio", "hypo", "wire"] if UID == 'p1' else ["istio", "hypo", "wire"]

RATES = {
    "istio": {
        "boutique": 50,
        "reservation": 1000,
        "social": 1500
    },
    "hypo": {
        "boutique": 50,
        "reservation": 1000,
        "social": 1500
    },
    "devbest": {
        "boutique": 100,
        "reservation": 1000,
        "social": 1500
    },
    "wire": {
        "boutique": 50,
        "reservation": 1000,
        "social": 1500
    }
}

initial_mem_u1 = {
    "istio": 8000,
    "hypo": 8000,
    "wire": 8200,
}
initial_mem_u2 = {
    "istio": 8000,
    "hypo": 8000,
    "wire": 8000,
}
initial_mem = initial_mem_u1 if UID == 'p1' else initial_mem_u2

num_applications = len(APPS)

# Plot formatting
plt.rcParams['text.usetex'] = True  #Let TeX do the typsetting
plt.rcParams['font.size'] = 14
# plt.rcParams['text.latex.preamble'] = [
#     r'\usepackage{sansmath}', r'\sansmath'
# ]  #Force sans-serif math mode (for axes labels)
plt.rcParams['font.family'] = 'sans-serif'  # ... for regular text
plt.rcParams[
    'font.sans-serif'] = 'Computer Modern Sans serif'  # Choose a nice font here

colors = {
    'istio': '#D5E8D4',
    'hypo': '#FFE6CC',
    'devbest': '#F8CECC',
    'wire': '#DAE8FC'
}
edge_colors = {
    'istio': '#82B366',
    'hypo': '#D79B00',
    'devbest': '#B85450',
    'wire': '#6C8EBF'
}
hatches = {
    'istio': '//', 
    'hypo': '..', 
    'devbest': '\\\\',
    'wire': '++'
}

label1 = mpatches.Patch(facecolor=colors['istio'],
                        edgecolor=edge_colors['istio'],
                        hatch=hatches['istio'],
                        label=LABELS['istio'])
label2 = mpatches.Patch(facecolor=colors['hypo'],
                        edgecolor=edge_colors['hypo'],
                        hatch=hatches['hypo'],
                        label=LABELS['hypo'])
label3 = mpatches.Patch(facecolor=colors['devbest'],
                        edgecolor=edge_colors['devbest'],
                        hatch=hatches['devbest'],
                        label=LABELS['devbest'])
label4 = mpatches.Patch(facecolor=colors['wire'],
                        edgecolor=edge_colors['wire'],
                        hatch=hatches['wire'],
                        label=LABELS['wire'])

mPatches = {
    'istio': label1,
    'hypo': label2,
    'devbest': label3,
    'wire': label4
}


def get_stats(type):
    stats = {}
    for appl in APPS:
        # Get the directory for the application
        stats[appl] = {}
        for dir, _, _ in os.walk(PATH):
            dirname = dir.split('/')[-1]
            if '-' not in dirname:
                continue

            app = dirname.split('-')[0]
            mesh = dirname.split('-')[1]
            rate = int(dirname.split('-')[2])

            if app == appl and mesh in MESH and rate == RATES[mesh][appl]:
                all_stats = []
                for n in range(NODES):
                    with open(
                            os.path.join(dir, 'stats_{0}_{1}_{2}.pkl'.format(appl, mesh, n)),
                            'rb') as f:
                        stats_node = pkl.load(f)
                        if len(all_stats) == 0:
                            all_stats = stats_node[type]
                        else:
                            all_stats = np.sum(
                                [all_stats, stats_node[type]], axis=0)
                stats[appl][mesh] = all_stats

        # Get the minimum length of the stats
        min_len = 100000
        for _, s in stats[appl].items():
            min_len = min(min_len, len(s))

        # Subtract the initial memory usage
        if type == 'used':
            for m in MESH:
                if m not in stats[appl]:
                    continue
                stats[appl][m] = np.subtract(stats[appl][m], initial_mem[m])

        offset = min_len - 60
        if min_len > 60:
            stats[appl] = {
                k: v[5+offset:min_len] for k, v in stats[appl].items()
            }
        else:
            stats[appl] = {k: v[5:min_len] for k, v in stats[appl].items()}

        # Print the analysis
        # print(appl, type)
        for m in MESH:
            if m not in stats[appl]:
                continue
            # print(m, np.mean(stats[appl][m]))

            # Print how many times the mean is higher compared to wire
            if m != 'wire':
                print(m, 1 - np.mean(stats[appl]['wire']) / np.mean(stats[appl][m]))

    return stats


def add_boxplot(ax, stats_data, ylabel):
    # Three box plots for each application. Each box plot has 3 boxes for each service mesh.
    # Color each service mesh differently.
    for m in MESH:
        stats = []
        # Get the CPU usage for each application
        for appl in APPS:
            stats.append(stats_data[appl][m])

        # Plot the box plots
        j = MESH.index(m)
        ax.boxplot(
            stats,
            positions=[i * 3 + j * 0.5 + 1 for i in range(len(APPS))],
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
        ax.set_xticks([i * 3 + 1.75 for i in range(len(APPS))])
        ax.set_xticklabels(APPS, fontsize=22)

        if ylabel == 'Memory Usage (MB)':
            ax.set_yticks([4800, 5000, 5200, 5400, 5600, 5800])
        ax.set_yticklabels([int(y) for y in ax.get_yticks()], fontsize=22)

        ax.set_ylabel(ylabel, fontsize=24)
        ax.set_xlabel('Applications', fontsize=24)

        # Set limit [10, 40] for cpu usage and [10500, 11500] for memory usage
        if ylabel == 'CPU Usage (\%)':
            ax.set_ylim([0, 50])


# Write a function add_barplot(ax, stats_data, ylabel) that adds a bar plot to the axis ax.
# Similar to the box plot, but with a bar plot.
def add_barplot(ax, stats_data, ylabel):
    A = 2.5
    B = 0.5
    C = 1.5

    # Three box plots for each application. Each box plot has 3 boxes for each service mesh.
    # Color each service mesh differently.
    for m in MESH:
        stats = []
        # Get the CPU usage for each application
        no_data = False
        for appl in APPS:
            if m not in stats_data[appl]:
                no_data = True
                break
            stats.append(stats_data[appl][m])

        if no_data:
            continue

        # Plot the box plots
        j = MESH.index(m)
        ax.bar(
            [i * A + j * B + 1 for i in range(len(APPS))],
            [np.mean(s) for s in stats],
            yerr=[np.std(s) for s in stats],
            capsize=5,
            width=0.4,
            color=colors[m],
            edgecolor=edge_colors[m],
            hatch=hatches[m])

        # Set the xticks
        ax.set_xticks([i * A + C for i in range(len(APPS))])
        ax.set_xticklabels([a.capitalize() for a in APPS], fontsize=24)

        if ylabel == 'Memory (GB)':
            ax.set_yticks(np.arange(0, 4, 1) * 1000)
            ax.set_yticklabels([str(int(y/1000)) for y in ax.get_yticks()], fontsize=22)
        else:
            ax.set_yticklabels([int(y) for y in ax.get_yticks()], fontsize=24)

        # ax.set_ylabel(ylabel, fontsize=24)
        ax.set_title(ylabel, fontsize=26)
        ax.set_xlabel('Applications', fontsize=26)

        # Set limit [10, 40] for cpu usage and [10500, 11500] for memory usage
        if ylabel == 'CPU Usage (\%)':
            ax.set_ylim([0, 50])


# Make a boxplot for CPU usage
figsize = (10, 4)
fig = plt.figure(figsize=(figsize[0], figsize[1]))

# Plot CPU usage
ax = fig.add_subplot(121)
cpu_stats = get_stats('cpu')
# add_boxplot(ax, cpu_stats, 'CPU Usage (\%)')
add_barplot(ax, cpu_stats, 'CPU Usage (\%)')

# Plot memory usage
ax = fig.add_subplot(122)
mem_stats = get_stats('used')
# add_boxplot(ax, mem_stats, 'Memory (MB)')
add_barplot(ax, mem_stats, 'Memory (GB)')

fig.legend(handles=[label1, label2, label4],
        loc='center',
        ncol=3,
        frameon=False,
        bbox_to_anchor=(0.5, 0.93),
        bbox_transform=fig.transFigure,
        fontsize=28)

# Save the plot
plt.tight_layout()
plt.subplots_adjust(top=0.76, bottom=0.2, left=0.05, right=0.98)
plt.savefig('cpumem_comp_{0}.png'.format(UID))
plt.savefig('cpumem_comp_{0}.pdf'.format(UID))
