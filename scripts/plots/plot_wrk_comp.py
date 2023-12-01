"""
Plot a median and tail latencies to compare the latencies between vanilla, istio and nginx.
Arguments:
1: the path to the folder containing the latency files.
"""

import os
import sys
import numpy as np
import matplotlib.pyplot as plt

# Parse the wrk output to get the percentile latencies
def get_latencies(content):
    latencies = {}
    cnt = 0
    for line in content:
        if len(line) < 2:
            continue

        if line[0] == '50.000%':
            if line[1][-2:] == 'ms':
                latencies['50'] = float(line[1][:-2])
            elif line[1][-1:] == 'm':
                latencies['50'] = float(line[1][:-1]) * 1000 * 60
            elif line[1][-1:] == 's':
                latencies['50'] = float(line[1][:-1]) * 1000
            else:
                print("Error: unknown unit for latency", line[1])
            cnt += 1
        elif line[0] == '90.000%':
            if line[1][-2:] == 'ms':
                latencies['90'] = float(line[1][:-2])
            elif line[1][-1:] == 'm':
                latencies['90'] = float(line[1][:-1]) * 1000 * 60
            elif line[1][-1:] == 's':
                latencies['90'] = float(line[1][:-1]) * 1000
            else:
                print("Error: unknown unit for latency", line[1])
            cnt += 1
        elif line[0] == '99.000%':
            if line[1][-2:] == 'ms':
                latencies['99'] = float(line[1][:-2])
            elif line[1][-1:] == 'm':
                latencies['99'] = float(line[1][:-1]) * 1000 * 60
            elif line[1][-1:] == 's':
                latencies['99'] = float(line[1][:-1]) * 1000
            else:
                print("Error: unknown unit for latency", line[1])
            cnt += 1
        elif line[0] == '99.900%':
            if line[1][-2:] == 'ms':
                latencies['99.9'] = float(line[1][:-2])
            elif line[1][-1:] == 'm':
                latencies['99.9'] = float(line[1][:-1]) * 1000 * 60
            elif line[1][-1:] == 's':
                latencies['99.9'] = float(line[1][:-1]) * 1000
            else:
                print("Error: unknown unit for latency", line[1])
            cnt += 1
        elif line[0] == '99.990%':
            if line[1][-2:] == 'ms':
                latencies['99.99'] = float(line[1][:-2])
            elif line[1][-1:] == 'm':
                latencies['99.99'] = float(line[1][:-1]) * 1000 * 60
            elif line[1][-1:] == 's':
                latencies['99.99'] = float(line[1][:-1]) * 1000
            else:
                print("Error: unknown unit for latency", line[1])
            cnt += 1
        
        if cnt == 5:
            break
    return latencies

if __name__ == '__main__':
    if len(sys.argv) < 3:
        print("Usage: python3 plot_wrk_comp.py <path> <application>")
        sys.exit(1)

    path = sys.argv[1]
    appl = sys.argv[2]

    with open(os.path.join(path, 'time_{0}_plain.run'.format(appl))) as f:
        content = f.readlines()
    content_plain = [x.strip().split() for x in content]
    latencies_plain = get_latencies(content_plain)

    with open(os.path.join(path, 'time_{0}_nginx.run'.format(appl))) as f:
        content = f.readlines()
    content_nginx = [x.strip().split() for x in content]
    latencies_nginx = get_latencies(content_nginx)

    with open(os.path.join(path, 'time_{0}_istio.run'.format(appl))) as f:
        content = f.readlines()
    content_istio = [x.strip().split() for x in content]
    latencies_istio = get_latencies(content_istio)

    pct_to_plot = ['50', '90', '99', '99.9', '99.99']

    # Pct wise ratio of latencies between nginx and plain and istio and plain
    for pct in pct_to_plot:
        print("Nginx: {:.2f}".format(latencies_nginx[pct] / latencies_plain[pct]))
        print("Istio: {:.2f}".format(latencies_istio[pct] / latencies_plain[pct]))

    # Plot formatting
    plt.rcParams['text.usetex'] == True
    plt.rcParams['font.family'] = ['sans-serif']  # ... for regular text
    plt.rcParams['font.sans-serif'] = ['Helvetica'] + plt.rcParams['font.sans-serif'] # Choose a nice font here

    # Bar plot to compare the various percentile of latencies
    fig = plt.figure(figsize=(10, 4.4))  # 6.4:4.8
    ax = fig.add_subplot(111)

    xs = np.arange(len(pct_to_plot))
    ax.bar(xs - 0.2,
        [latencies_plain[pct] for pct in pct_to_plot],
        width=0.2,
        edgecolor='black',
        hatch='//',
        label='Without Service Mesh')
    ax.bar(xs,
        [latencies_nginx[pct] for pct in pct_to_plot],
        width=0.2,
        edgecolor='black',
        hatch='..',
        label='With Nginx')
    ax.bar(xs + 0.2,
        [latencies_istio[pct] for pct in pct_to_plot],
        width=0.2,
        edgecolor='black',   
        hatch='\\\\',
        label='With Istio')

    yticks = np.linspace(0, 40000, 9)

    ax.set_yticks(yticks)
    ax.set_yticklabels(yticks, fontsize=16)
    ax.set_xticks(xs, pct_to_plot)
    ax.set_xticklabels(pct_to_plot, fontsize=16)
    ax.set_xlabel("Percentile", fontsize=20, fontweight='bold')
    ax.set_ylabel("Latency (ms)", fontsize=20, fontweight='bold')
    ax.legend(loc='upper left', fontsize=16)

    plt.subplots_adjust(left=0.15, right=0.95, top=0.95, bottom=0.14)
    plt.savefig(os.path.join(path, 'comparison_wrk.png'))
    plt.savefig(os.path.join(path, 'comparison_wrk.pdf'))