"""
Plot median and tail latencies to compare the latencies between plain, istio and nginx
for multiple applications.
"""

import os
import sys
import numpy as np
import matplotlib.pyplot as plt
import matplotlib.patches as mpatches

from plot_wrk_comp import get_latencies
from matplotlib.ticker import ScalarFormatter

if len(sys.argv) < 2:
    print("Usage: python3 plot_motivation_latency.py <path> <low/high>")
    sys.exit(1)


path = sys.argv[1]
LOWLOAD = sys.argv[2] == 'low'

applications = ['bookinfo', 'boutique', 'reservation']
latencies = []

for appl in applications:
    appl_dir = appl
    if LOWLOAD:
        appl_dir += '-lowload'

    with open(os.path.join(path, appl_dir,
                           'time_{0}_plain.run'.format(appl))) as f:
        content = f.readlines()
    content_plain = [x.strip().split() for x in content]
    latencies_plain = get_latencies(content_plain)

    with open(os.path.join(path, appl_dir,
                           'time_{0}_nginx.run'.format(appl))) as f:
        content = f.readlines()
    content_nginx = [x.strip().split() for x in content]
    latencies_nginx = get_latencies(content_nginx)

    with open(os.path.join(path, appl_dir,
                           'time_{0}_istio.run'.format(appl))) as f:
        content = f.readlines()
    content_istio = [x.strip().split() for x in content]
    latencies_istio = get_latencies(content_istio)

    latencies.append([latencies_plain, latencies_nginx, latencies_istio])

# Plot percentiles
pct_to_plot = ['50', '99']

# Plot formatting
plt.rcParams['text.usetex'] = True  #Let TeX do the typsetting
plt.rcParams['font.size'] = 14
plt.rcParams['text.latex.preamble'] = [
    r'\usepackage{sansmath}', r'\sansmath'
]  #Force sans-serif math mode (for axes labels)
plt.rcParams['font.family'] = 'sans-serif'  # ... for regular text
plt.rcParams[
    'font.sans-serif'] = 'Computer Modern Sans serif'  # Choose a nice font here

fig = plt.figure(figsize=(8.8, 4.2))
ax = fig.add_subplot(111)

a_val = 1.0
colors = ['#D5E8D4', '#FFE6CC', '#F8CECC']
edge_colors = ['#82B366', '#D79B00', '#B85450']
label1 = mpatches.Patch(facecolor=colors[0],
                        edgecolor=edge_colors[0],   
                        alpha=a_val,
                        hatch='//',
                        label='No Service Mesh')
label2 = mpatches.Patch(facecolor=colors[1],
                        edgecolor=edge_colors[1],
                        alpha=a_val,
                        hatch='..',
                        label='Nginx Mesh')
label3 = mpatches.Patch(facecolor=colors[2],
                        edgecolor=edge_colors[2],
                        alpha=a_val,
                        hatch=r'\\\\',
                        label='Istio Mesh')

# Make a bar plot. The plot should have 3 bars for each application, one for each
# service mesh. Each bar should have 2 sub-bars, one for median and one for 99th
# percentile.
xs_app = np.arange(len(applications))
width = 0.25

# Plot one bar for median at xs and one for 99th percentile at xs * 2
for i, p in enumerate(pct_to_plot):
    ax.bar(xs_app * len(pct_to_plot) + i - width,
           [latencies[j][0][p] for j in range(len(applications))],
           width=width,
           edgecolor=edge_colors[0],
           color=colors[0],
           hatch='//')
    ax.bar(xs_app * len(pct_to_plot) + i,
           [latencies[j][1][p] for j in range(len(applications))],
           width=width,
           edgecolor=edge_colors[1],
           color=colors[1],
           hatch='..')
    ax.bar(xs_app * len(pct_to_plot) + i + width,
           [latencies[j][2][p] for j in range(len(applications))],
           width=width,
           edgecolor=edge_colors[2],
           color=colors[2],
           hatch='\\\\')

xs = np.arange(len(applications) * len(pct_to_plot))
ax.set_xticks(xs)

labels = []
for appl in applications:
    for p in pct_to_plot:
        labels.append(appl + '\n' + p + 'th')
ax.set_xticklabels(labels, fontsize=20)

# Y-axis log scale
if LOWLOAD:
    ys = [10, 100, 1000]
    ax.set_yscale('log')
    ax.set_yticks(ys)
    ax.set_yticklabels([str(y) for y in ys])
    ax.tick_params(axis='y', labelsize=20)
else:
    ys = [1000, 5000, 10000, 50000]
    ax.set_yscale('log')
    ax.set_yticks(ys)
    ax.set_yticklabels([str(y) for y in ys])
    ax.tick_params(axis='y', labelsize=20)

ax.set_xlabel('Applications', fontsize=22)
ax.set_ylabel('Latency (ms)', fontsize=22)
ax.legend(handles=[label1, label2, label3],
          loc='center',
          ncol=3,
          bbox_to_anchor=(0.05, 1.05, 0.8, 0.15),
          fontsize=20)

plt.subplots_adjust(left=0.12, right=0.98, top=0.85, bottom=0.25)
if not LOWLOAD:
    plt.savefig(os.path.join(path, 'mot-latency-heavy.pdf'))
    plt.savefig(os.path.join(path, 'mot-latency-heavy.png'))
else:
    plt.savefig(os.path.join(path, 'mot-latency-light.pdf'))
    plt.savefig(os.path.join(path, 'mot-latency-light.png'))
