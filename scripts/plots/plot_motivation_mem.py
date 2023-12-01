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
    print("Usage: python3 plot_motivation_mem.py <path> <low/high>")
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

mem_stats = []

for appl in applications:
    appl_dir = appl
    if LOWLOAD:
        appl_dir += '-lowload'

    with open(os.path.join(path, appl_dir, 'stats_{0}_plain.pkl'.format(appl)),
              'rb') as f:
        stats_plain = pkl.load(f)

    with open(os.path.join(path, appl_dir, 'stats_{0}_nginx.pkl'.format(appl)),
              'rb') as f:
        stats_nginx = pkl.load(f)

    with open(os.path.join(path, appl_dir, 'stats_{0}_istio.pkl'.format(appl)),
              'rb') as f:
        stats_istio = pkl.load(f)

    # Get the minimum length of the stats
    print("Plain: {0}, Nginx: {1}, Istio: {2}".format(len(stats_plain['cpu']),
                                                      len(stats_nginx['cpu']),
                                                      len(stats_istio['cpu'])))
    min_len = min(len(stats_plain['cpu']), len(stats_nginx['cpu']),
                  len(stats_istio['cpu']))
    mem_stats.append([
        stats_plain['used'][5:min_len], stats_nginx['used'][5:min_len],
        stats_istio['used'][5:min_len]
    ])

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
fig = plt.figure(figsize=(5.6, 4.2))
ax = fig.add_subplot(111)

# Three box plots for each application. Each box plot has 3 boxes for each service mesh.
# Color each service mesh differently.
for j in range(3):
    stats = []
    # Get the CPU usage for each application
    for i, appl in enumerate(applications):
        stats.append(mem_stats[i][j])

    # Plot the box plots
    ax.boxplot(stats,
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

ys = [int(y) for y in np.linspace(4000, 5000, 6)]
ax.set_yticks(ys)
ax.set_yticklabels(ys)
ax.tick_params(axis='y', labelsize=20)

ax.set_ylabel('Mem Usage (in MB)', fontsize=22)
ax.legend(handles=[label1, label2, label3],
          loc='center',
          ncol=2,
          bbox_to_anchor=(0, 1.12, 0.8, 0.15),
          fontsize=20)

# Save the plot
plt.subplots_adjust(left=0.18, right=0.98, top=0.75, bottom=0.1)
if not LOWLOAD:
    plt.savefig(os.path.join(path, 'mot-mem-heavy.pdf'))
    plt.savefig(os.path.join(path, 'mot-mem-heavy.png'))
else:
    plt.savefig(os.path.join(path, 'mot-mem-light.pdf'))
    plt.savefig(os.path.join(path, 'mot-mem-light.png'))