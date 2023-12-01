"""
Plot the time consumption of policy placement module.
"""

import os
import sys
import numpy as np
import matplotlib.pyplot as plt
import matplotlib.patches as mpatches
from matplotlib.ticker import ScalarFormatter

# Read the terminal_output file
if len(sys.argv) < 2:
    print("Usage: python3 plot_placement.py <path>")
    sys.exit(1)

path = sys.argv[1]

# Get the time consumption
with open(os.path.join(path, 'terminal_output')) as f:
    content = f.readlines()
content = [x.strip().split(':') for x in content]

sizes = ['small', 'medium', 'large']

times = {}
additional_times = {}

for line in content:
    for size in sizes:
        if size in line[0]:
            name = size
            if 'dense' in line[0]:
                name = 'dense_' + size

            if 'Additional' in line[0]:
                l = line[1].strip().split(' ')
                additional_times[name] = [float(x) for x in l]
            else:
                times[name] = float(line[1].strip())

# Convert times to sec
for key in times:
    times[key] = times[key] / 1000
for key in additional_times:
    for i in range(len(additional_times[key])):
        additional_times[key][i] = additional_times[key][i] / 1000

print(times)
print(additional_times)

# Plot formatting
plt.rcParams['text.usetex'] = True  #Let TeX do the typsetting
plt.rcParams['font.size'] = 14
plt.rcParams['text.latex.preamble'] = [
    r'\usepackage{sansmath}', r'\sansmath'
]  #Force sans-serif math mode (for axes labels)
plt.rcParams['font.family'] = 'sans-serif'  # ... for regular text
plt.rcParams[
    'font.sans-serif'] = 'Computer Modern Sans serif'  # Choose a nice font here

# Plot the time consumption
fig = plt.figure(figsize=(4.8, 3.6))
ax = fig.add_subplot(111)

y = [times['small'], times['medium'], times['large']]
y_dense = [times['dense_small'], times['dense_medium'], times['dense_large']]

colors = ['#82B366', '#D79B00', '#B85450']
lines = ['o-', 'x--']

xs = [10, 20, 30]
ax.scatter(xs, y, marker='o', color=colors, s=50, label='Sparse Graph')
ax.scatter(xs, y_dense, marker='x', color=colors, s=100, label='Dense Graph')

ax.set_xticks(xs)
ax.set_xticklabels(['Small', 'Medium', 'Large'], fontsize=20)
ax.set_xlim(5, 35)

ys = [1, 5, 100, 500]
ax.set_yscale('log')
ax.set_yticks(ys)
ax.set_yticklabels([int(y) for y in ys])
ax.tick_params(axis='y', which='minor', left=False)
ax.tick_params(axis='y', labelsize=20)

ax.set_xlabel('Application Size', fontsize=22)
ax.set_ylabel('Time (s)', fontsize=22)
ax.yaxis.set_major_formatter(ScalarFormatter())

leg = ax.legend(loc='center',
                ncol=2,
                bbox_to_anchor=(-0.05, 1.12, 0.8, 0.15),
                frameon=False,
                handletextpad=0.1,
                columnspacing=0.5,
                fontsize=20)
leg.legendHandles[0].set_color('black')
leg.legendHandles[1].set_color('black')

plt.subplots_adjust(left=0.22, right=0.95, top=0.78, bottom=0.2)
plt.savefig(os.path.join(path, 'placement.pdf'))
plt.savefig(os.path.join(path, 'placement.png'))
plt.close()

# Plot the time taken for additional policies
fig = plt.figure(figsize=(5.2, 3.6))
ax = fig.add_subplot(111)

num_policies = [1, 5, 10, 15, 20]
ax.plot(num_policies,
        additional_times['medium'],
        lines[0],
        color=colors[1],
        label='Sparse Medium')
ax.plot(num_policies,
        additional_times['dense_medium'],
        lines[1],
        color=colors[1],
        label='Dense Medium')
ax.plot(num_policies,
        additional_times['large'],
        lines[0],
        color=colors[2],
        label='Sparse Large')
ax.plot(num_policies,
        additional_times['dense_large'],
        lines[1],
        color=colors[2],
        label='Dense Large')

ax.set_xticks(num_policies)
ax.set_xticklabels(num_policies, fontsize=20)

ys = np.linspace(0, 5, 6)
ax.set_yticks(ys)
ax.set_yticklabels([int(y * 1000) for y in ys])
ax.tick_params(axis='y', labelsize=20)

ax.set_xlabel('Number of Policies', fontsize=22)
ax.set_ylabel('Time (ms)', fontsize=22)

ax.legend(loc='center',
          ncol=2,
          frameon=False,
          columnspacing=0.5,
          bbox_to_anchor=(-0.05, 1.12, 0.95, 0.15),
          fontsize=20)

plt.subplots_adjust(left=0.2, right=0.95, top=0.78, bottom=0.2)
plt.savefig(os.path.join(path, 'placement-additional.pdf'))
plt.savefig(os.path.join(path, 'placement-additional.png'))
plt.close()