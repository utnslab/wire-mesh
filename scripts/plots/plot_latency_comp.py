"""
Plot median and tail latencies to compare the latencies between plain, istio and wire
for multiple applications.
"""

import os
import sys
import json
import numpy as np
import matplotlib.pyplot as plt
import matplotlib.patches as mpatches

from plot_wrk_comp import get_latencies
from matplotlib.ticker import ScalarFormatter

if len(sys.argv) < 2:
    print("Usage: python3 plot_latency_comp.py <path> <low/high>")
    sys.exit(1)

num_applications = 3
applications = ['bookinfo', 'boutique', 'reservation']
directory = sys.argv[1]
LOWLOAD = sys.argv[2] == 'low'

# Read eval_config.json to get directories for each application
current_dir = os.path.dirname(os.path.realpath(__file__))
json_file = os.path.join(current_dir, 'eval_config.json')

# Plot percentiles
pct_to_plot = ['50', '90', '99', '99.9']

with open(json_file) as f:
    results_data = json.load(f)

latencies = []
for appl in applications:
    # Get the directory for the application
    appl_dir = results_data[appl]['heavy']
    if LOWLOAD:
        appl_dir = results_data[appl]['light']

    # Get the percentile data
    plain_file = os.path.join(directory, appl_dir['plain'],
                              'time_{0}_plain.run'.format(appl))
    istio_file = os.path.join(directory, appl_dir['istio'],
                              'time_{0}_istio.run'.format(appl))
    wire_file = os.path.join(directory, appl_dir['wire'],
                             'time_{0}_wire.run'.format(appl))
    wire_best_file = os.path.join(directory, appl_dir['wire-best'],
                                  'time_{0}_wire.run'.format(appl))

    with open(plain_file) as f:
        content = f.readlines()
    content_plain = [x.strip().split() for x in content]
    latencies_plain = get_latencies(content_plain)

    with open(wire_file) as f:
        content = f.readlines()
    content_wire = [x.strip().split() for x in content]
    latencies_wire = get_latencies(content_wire)

    with open(wire_best_file) as f:
        content = f.readlines()
    content_wire_best = [x.strip().split() for x in content]
    latencies_wire_best = get_latencies(content_wire_best)

    with open(istio_file) as f:
        content = f.readlines()
    content_istio = [x.strip().split() for x in content]
    latencies_istio = get_latencies(content_istio)

    print(appl)
    for p in pct_to_plot:
        print('Istio/Wire: {0:.2f}'.format(latencies_istio[p] /
                                           latencies_wire[p]))
        print('Istio/Wire-best: {0:.2f}'.format(latencies_istio[p] /
                                                latencies_wire_best[p]))
    print()

    latencies.append([latencies_wire_best, latencies_wire, latencies_istio])

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
label1 = mpatches.Patch(facecolor=colors[0],
                        edgecolor=edge_colors[0],
                        alpha=a_val,
                        hatch='//',
                        label='Wire - Best Case')
label2 = mpatches.Patch(facecolor=colors[1],
                        edgecolor=edge_colors[1],
                        alpha=a_val,
                        hatch='..',
                        label='Wire - Worst Case')
label3 = mpatches.Patch(facecolor=colors[2],
                        edgecolor=edge_colors[2],
                        alpha=a_val,
                        hatch=r'\\\\',
                        label='Istio')

# Make a bar plot. The plot should have 3 bars for each application, one for each
# service mesh. Each bar should have 2 sub-bars, one for median and one for 99th
# percentile.
xs = np.arange(4)
width = 0.25

# Plot one bar for median at xs and one for 99th percentile at xs * 2
fig, axs = plt.subplots(1, num_applications, figsize=(14, 3.6), sharey=True)
for i in range(num_applications):
    axs[i].bar(xs - width, [latencies[i][0][p] for p in pct_to_plot],
               width=width,
               edgecolor=edge_colors[0],
               color=colors[0],
               hatch='//')
    axs[i].bar(xs, [latencies[i][1][p] for p in pct_to_plot],
               width=width,
               edgecolor=edge_colors[1],
               color=colors[1],
               hatch='..')
    axs[i].bar(xs + width, [latencies[i][2][p] for p in pct_to_plot],
               width=width,
               edgecolor=edge_colors[2],
               color=colors[2],
               hatch='\\\\')

    axs[i].set_xticks(xs)

    labels = []
    for p in pct_to_plot:
        labels.append(p + 'th')
    axs[i].set_xticklabels(labels, fontsize=20)

    axs[i].set_xlabel(applications[i], fontsize=22)

# Y-axis log scale
ys = [int(y) for y in np.linspace(0, 50000, 6)]
# ys = [1000, 5000, 10000, 50000]
if LOWLOAD:
    # axs[0].set_yscale('log')
    axs[0].set_yticks(ys)
    axs[0].set_yticklabels([str(y) for y in ys])
    axs[0].tick_params(axis='y', labelsize=20)
else:
    # axs[0].set_yscale('log')
    axs[0].set_yticks(ys)
    axs[0].set_yticklabels([str(y) for y in ys])
    axs[0].tick_params(axis='y', labelsize=20)

# ax.set_xlabel('Applications', fontsize=22)
axs[0].set_ylabel('Latency (ms)', fontsize=22)
axs[1].legend(handles=[label1, label2, label3],
              loc='center',
              ncol=3,
              frameon=False,
              bbox_to_anchor=(0.05, 1.025, 0.8, 0.15),
              fontsize=20)

plt.subplots_adjust(left=0.08, right=0.98, top=0.87, bottom=0.2, wspace=0.08)
if LOWLOAD:
    plt.savefig('latency-comp-light.pdf')
    plt.savefig('latency-comp-light.png')
else:
    plt.savefig('latency-comp-heavy.pdf')
    plt.savefig('latency-comp-heavy.png')
