"""
Make box plots for CPU and memory usage for each application and service mesh.
"""

import os
import sys
import numpy as np
import pickle as pkl
import matplotlib.pyplot as plt
import matplotlib.patches as mpatches

if len(sys.argv) < 2:
    print("Usage: python3 plot_motivation_memcpu.py <path> <low/high>")
    sys.exit(1)

path = sys.argv[1]
LOWLOAD = sys.argv[2] == 'low'

applications = ['bookinfo', 'boutique', 'reservation']

colors = ['#D5E8D4', '#FFE6CC', '#F8CECC']
edge_colors = ['#82B366', '#D79B00', '#B85450']
hatches = ['//', '..', '\\\\']

label1 = mpatches.Patch(facecolor=colors[0],
                        edgecolor=edge_colors[0],
                        hatch=hatches[0],
                        label='No Service Mesh')
label2 = mpatches.Patch(facecolor=colors[1],
                        edgecolor=edge_colors[1],
                        hatch=hatches[1],
                        label='Nginx Mesh')
label3 = mpatches.Patch(facecolor=colors[2],
                        edgecolor=edge_colors[2],
                        hatch=hatches[2],
                        label='Istio Mesh')


def get_stats(type):
    stats = []

    for appl in applications:
        appl_dir = appl
        if LOWLOAD:
            appl_dir += '-lowload'

        with open(os.path.join(path, appl_dir,
                             'stats_{0}_plain.pkl'.format(appl)), 'rb') as f:
            stats_plain = pkl.load(f)

        with open(os.path.join(path, appl_dir,
                             'stats_{0}_nginx.pkl'.format(appl)), 'rb') as f:
            stats_nginx = pkl.load(f)

        with open(os.path.join(path, appl_dir,
                             'stats_{0}_istio.pkl'.format(appl)), 'rb') as f:
            stats_istio = pkl.load(f)

        # Get the minimum length of the stats
        print("Plain: {0}, Nginx: {1}, Istio: {2}".format(
            len(stats_plain[type]), len(stats_nginx[type]),
            len(stats_istio[type])))
        min_len = min(len(stats_plain[type]), len(stats_nginx[type]),
                      len(stats_istio[type]))
        stats.append([
            stats_plain[type][5:min_len], stats_nginx[type][5:min_len],
            stats_istio[type][5:min_len]
        ])

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
        ax.set_yticklabels(ax.get_yticks(), fontsize=20)

        ax.set_ylabel(ylabel, fontsize=22)
        ax.set_xlabel('Applications', fontsize=22)


# Plot formatting
plt.rcParams['text.usetex'] = True  #Let TeX do the typsetting
plt.rcParams['font.size'] = 14
plt.rcParams['text.latex.preamble'] = [
    r'\usepackage{sansmath}', r'\sansmath'
]  #Force sans-serif math mode (for axes labels)
plt.rcParams['font.family'] = 'sans-serif'  # ... for regular text
plt.rcParams[
    'font.sans-serif'] = 'Computer Modern Sans serif'  # Choose a nice font here

# Make a boxplot for CPU usage
fig = plt.figure(figsize=(10.2, 4))

# Plot CPU usage
ax = fig.add_subplot(121)
cpu_stats = get_stats('cpu')
add_boxplot(ax, cpu_stats, 'CPU Usage (\%)')

# Plot memory usage
ax = fig.add_subplot(122)
mem_stats = get_stats('used')
add_boxplot(ax, mem_stats, 'Memory Usage (MB)')

fig.legend(handles=[label1, label2, label3],
           loc='center',
           ncol=3,
           bbox_to_anchor=(0.5, 0.93),
           bbox_transform=fig.transFigure,
           fontsize=20)

# Save the plot
plt.tight_layout()
plt.subplots_adjust(top=0.85, bottom=0.18)
if not LOWLOAD:
    plt.savefig(os.path.join(path, 'mot-memcpu-heavy.pdf'))
    plt.savefig(os.path.join(path, 'mot-memcpu-heavy.png'))
else:
    plt.savefig(os.path.join(path, 'mot-memcpu-light.pdf'))
    plt.savefig(os.path.join(path, 'mot-memcpu-light.png'))
