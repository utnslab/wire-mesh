"""
Plot latency vs throughput graph for a specific application
"""

import os
import sys
import numpy as np

from plot_wrk_comp import get_latencies
import matplotlib.pyplot as plt
from scipy.interpolate import interp1d

if len(sys.argv) < 2:
    print("Usage: python3 plot_tput_latency.py <path> <list of apps>")
    sys.exit(1)

MESH = ["istio", "hypo", "devbest", "wire"]
PATH = sys.argv[1]
APPS = sys.argv[2:]

UID = PATH.split('/')[-1]
LABELS = {
    'istio': 'Istio',
    'hypo': 'Istio+DG',
    'devbest': 'Manual Best',
    'wire': 'Wire'
}
PERC = ['99']

# Interpolation function
KIND = 'zero'

# Plot formatting
plt.rcParams['text.usetex'] = True  #Let TeX do the typsetting
plt.rcParams['font.size'] = 14
plt.rcParams['text.latex.preamble'] = [
    r'\usepackage{sansmath}', r'\sansmath'
]  #Force sans-serif math mode (for axes labels)
plt.rcParams['font.family'] = 'sans-serif'  # ... for regular text
plt.rcParams[
    'font.sans-serif'] = 'Computer Modern Sans serif'  # Choose a nice font here

# Plot the graph
fig = plt.figure(figsize=(15, 3.6))

colors = ['#82B366', '#D79B00', '#B85450', '#6C8EBF', '#9673A6', '#D6B656']
markers = ['o', 'P', '^', 's', 'v', '^']
styles = ['-', '--', '-.', 'dotted']

for a in range(len(APPS)):
    ax = fig.add_subplot(1, len(APPS), a+1)

    APP = APPS[a]
    if APP == 'reservation':
        RATES = [100, 400, 600, 800, 1000, 1500, 2000, 2500, 3000, 4000, 5000, 6000]
    elif APP == 'social':
        RATES = [100, 400, 600, 800, 1000, 1500, 2000, 2500, 3000, 3500, 4000]
    elif APP == 'boutique':
        RATES = [50, 80, 100, 120, 150, 200, 250, 300, 400, 450, 500]


    # Construct the latencies object to store values
    latencies = []
    for m in MESH:
        latencies_dict = {}
        for p in PERC:
            latencies_dict[p] = {}
        latencies.append(latencies_dict)

    # Walk through the directory and get the latencies
    for dir, _, _ in os.walk(PATH):
        dirname = dir.split('/')[-1]
        if '-' not in dirname:
            continue

        app = dirname.split('-')[0]
        mesh = dirname.split('-')[1]
        rate = int(dirname.split('-')[2])

        if app == APP and mesh in MESH and rate in RATES:
            # Get the percentile data
            file = os.path.join(dir, 'time_{0}_{1}_{2}.run'.format(app, rate, mesh))
            
            # Check if the file exists
            if not os.path.exists(file):
                continue
            
            with open(file) as f:
                content = f.readlines()
            content = [x.strip().split() for x in content]
            latencies_dict = get_latencies(content)

            for p in PERC:
                latencies[MESH.index(mesh)][p][rate] = latencies_dict[p]

    # # Print the latencies
    # for m in MESH:
    #     print(m)
    #     for p in PERC:
    #         print(p, latencies[MESH.index(m)][p])
    
    # # Compute the smoothed curve
    # plot_latencies = []
    # for m in MESH:
    #     plot_latencies.append({})
    #     if UID == 'p2' and m == 'devbest':
    #         continue
    #     for p in PERC:
    #         plot_latencies[MESH.index(m)][p] = {}
    #         latency = []
    #         rates = []
    #         for rate in RATES:
    #             if rate in latencies[MESH.index(m)][p]:
    #                 latency.append(latencies[MESH.index(m)][p][rate])
    #                 rates.append(rate)
    #         xnew = np.linspace(min(rates), max(rates), 30)
    #         print(m, p, rates, latency)
    #         spl = interp1d(rates, latency, kind=KIND)
    #         plot_latencies[MESH.index(m)][p] = {x: y for x, y in zip(xnew, spl(xnew))}
    plot_latencies = latencies

    # Print the Analysis
    print("Analysis:")
    # Print how many times higher the latencies are compared to the last element in array.
    for m in MESH[:-1]:
        rate = RATES[0]
        for p in PERC:
            if rate not in latencies[MESH.index(m)][p]:
                continue
            print(m, p, latencies[MESH.index(m)][p][rate] / latencies[MESH.index('wire')][p][rate])

    for m in range(len(MESH)):
        # Ignore if plot_latencies[m] is empty
        if not plot_latencies[m]:
            continue
        for p in PERC:
            rates = sorted(list(plot_latencies[m][p].keys()))
            latency = [plot_latencies[m][p][rate] for rate in rates]
            style = '-'
            if len(rates) == 0:
                continue
            ax.plot(rates, latency, label=LABELS[MESH[m]], linewidth=6,
                    color=colors[m], linestyle=styles[m])
            # ax.plot(rates, latency, label=LABELS[MESH[m]]+' '+p+'p', linewidth=4,
            #         color=colors[m], linestyle=styles[m])


    ax.set_xlabel('Throughput (req/s)')

    if APP == 'reservation':
        xs = [int(x) for x in np.linspace(0, 4000, 5)] if UID == 'p1' else [int(x) for x in np.linspace(0, 3000, 4)]
        xlim = [50, 4200] if UID == 'p1' else [50, 3200]
        ylim = [1, 1000]
    elif APP == 'social':
        xs = [int(x) for x in np.linspace(0, 4000, 5)] if UID == 'p1' else [int(x) for x in np.linspace(0, 3000, 4)]
        xlim = [50, 4200] if UID == 'p1' else [50, 3200]
        ylim = [1, 1000]
    elif APP == 'boutique':
        xs = [int(x) for x in np.linspace(0, 500, 6)] if UID == 'p1' else [int(x) for x in np.linspace(0, 300, 7)]
        xlim = [40, 520] if UID == 'p1' else [40, 260]
        ylim = [1, 1000]

    ax.set_xticks(xs)
    ax.set_xticklabels([str(x) for x in xs], fontsize=24)
    ax.set_xlim(xlim)
    ax.set_xlabel('Client Request Rate', fontsize=24)

    ys = np.linspace(0, 1000, 6)
    # ys = [0, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000]
    # ax.set_yscale('log', base=50)
    ax.set_yticks(ys)
    ax.set_yticklabels([str(int(y)) for y in ys], fontsize=24)
    ax.set_ylim(ylim)
    ax.set_ylabel('99p Latency (ms)', fontsize=24)
    ax.grid(True, which='major', axis='y', linestyle='--', linewidth=1)

    ax.set_title(APP.capitalize(), fontsize=26)

lines_labels = [ax.get_legend_handles_labels()]
lines, labels = [sum(lol, []) for lol in zip(*lines_labels)]
# fig.legend(lines, labels)

ncol = 4 if UID == 'p1' else 3
fig.legend(lines, labels, loc='center',
        ncol=ncol,
        frameon=False,
        columnspacing=0.5,
        bbox_to_anchor=(0.05, 1, 0.95, 0.15),
        fontsize=28)

fig.tight_layout()
plt.minorticks_off()
plt.subplots_adjust(left=0.08, right=0.98, top=0.9, bottom=0.15)

# plt.show()
plt.savefig('tput_latency_{0}.png'.format(UID), bbox_inches='tight')
plt.savefig('tput_latency_{0}.pdf'.format(UID), bbox_inches='tight')
plt.close()