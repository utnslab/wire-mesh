"""
Plot latency vs throughput graph for a specific application
"""

import os
import sys
import numpy as np

from plot_wrk_comp import get_latencies
import matplotlib.pyplot as plt
import matplotlib.ticker as ticker

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
PERC = ['50', '90']

for APP in APPS:
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

    # Print the latencies
    for m in MESH:
        print(m)
        for p in PERC:
            print(p, latencies[MESH.index(m)][p])

    # Print the Analysis
    print("Analysis:")
    # Print how many times higher the latencies are compared to the last element in array.
    for m in MESH:
        print(m)
        for p in PERC:
            rates = sorted(list(latencies[MESH.index(m)][p].keys()))
            latency = [latencies[MESH.index(m)][p][rate] for rate in rates]
            print(p, [round(x/latency[-1], 2) for x in latency])


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
    fig = plt.figure(figsize=(8, 4.8))
    ax = fig.add_subplot(111)

    colors = ['#82B366', '#D79B00', '#B85450', '#6C8EBF', '#9673A6', '#D6B656']
    markers = ['o', 'P', '^', 's', 'v', '^']

    for m in range(len(MESH)):
        if len(latencies[m]['50']) == 0:
            # Ignore if devbest is not there.
            continue
        for p in PERC:
            rates = sorted(list(latencies[m][p].keys()))
            latency = [latencies[m][p][rate] for rate in rates]
            style = '-' if p == '50' else 'dotted'
            ax.plot(rates, latency, label=LABELS[MESH[m]]+' '+p+'p', linewidth=4,
                    color=colors[m], marker=markers[m], markersize=12, linestyle=style)


    ax.set_xlabel('Throughput (req/s)')

    if APP == 'reservation':
        xs = [int(x) for x in np.linspace(0, 4000, 9)] if UID == 'p1' else [int(x) for x in np.linspace(0, 3500, 8)]
        xlim = [50, 4200] if UID == 'p1' else [50, 3200]
        ylim = [1, 5000]
    elif APP == 'social':
        xs = [int(x) for x in np.linspace(0, 4000, 9)] if UID == 'p1' else [int(x) for x in np.linspace(0, 3500, 8)]
        xlim = [50, 4200] if UID == 'p1' else [50, 3700]
        ylim = [1, 5000]
    elif APP == 'boutique':
        xs = [int(x) for x in np.linspace(0, 500, 11)] if UID == 'p1' else [int(x) for x in np.linspace(0, 300, 7)]
        xlim = [40, 520] if UID == 'p1' else [40, 300]
        ylim = [1, 5000]

    print(xs)
    ax.set_xticks(xs)
    ax.set_xticklabels([str(x) for x in xs], fontsize=20)
    ax.set_xlim(xlim)
    ax.set_xlabel('Throughput (req/s)', fontsize=22)

    # ys = np.logspace(0, 4, 5)
    ys = [0, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000]
    print(ys)
    ax.set_yscale('log', base=50)
    ax.set_yticks(ys)
    ax.set_yticklabels([str(y) for y in ys], fontsize=20)
    ax.set_ylim(ylim)
    ax.set_ylabel('Latency (ms)', fontsize=22)

    ncol = 4 if UID == 'p1' else 3
    ax.legend(loc='center',
            ncol=ncol,
            frameon=False,
            columnspacing=0.5,
            bbox_to_anchor=(0, 1.1, 0.95, 0.15),
            fontsize=18)
    ax.grid(True, which='major', axis='y', linestyle='--', linewidth=1)

    fig.tight_layout()
    plt.minorticks_off()
    plt.subplots_adjust(left=0.12, right=0.98, top=0.85, bottom=0.25)

    # plt.show()
    plt.savefig('tput_latency_{0}_{1}.png'.format(APP, UID), bbox_inches='tight')
    plt.savefig('tput_latency_{0}_{1}.pdf'.format(APP, UID), bbox_inches='tight')
    plt.close()