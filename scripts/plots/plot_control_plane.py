"""
Make box plots to show the fraction of removed sidecars and removed hotspots.
"""

import os
import sys
import json
import numpy as np
import matplotlib.pyplot as plt
import matplotlib.patches as mpatches

if len(sys.argv) < 2:
    print("Usage: python3 plot_control_plane.py <path>")
    sys.exit(1)

PATH = sys.argv[1]

# Read "removed_bestcase.json" and "removed_worstcase.json".
removed_bestcase = {}
removed_worstcase = {}

# Plot formatting
plt.rcParams["text.usetex"] = True  # Let TeX do the typsetting
plt.rcParams["font.size"] = 14
# plt.rcParams['text.latex.preamble'] = [
#     r'\usepackage{sansmath}', r'\sansmath'
# ]  #Force sans-serif math mode (for axes labels)
plt.rcParams["font.family"] = "sans-serif"  # ... for regular text
plt.rcParams["font.sans-serif"] = (
    "Computer Modern Sans serif"  # Choose a nice font here
)

colors = ["#D5E8D4", "#FFE6CC", "#F8CECC"]
edge_colors = ["#82B366", "#D79B00", "#B85450"]
hatches = ["//", "..", "\\\\"]

label1 = mpatches.Patch(
    facecolor=colors[0], edgecolor=edge_colors[0], hatch=hatches[0], label="P1"
)
label2 = mpatches.Patch(
    facecolor=colors[1], edgecolor=edge_colors[1], hatch=hatches[1], label="P1+P2"
)


with open(os.path.join(PATH, "removed_bestcase.json"), "r") as f:
    removed_bestcase = json.load(f)

with open(os.path.join(PATH, "removed_worstcase.json"), "r") as f:
    removed_worstcase = json.load(f)

# Find the median number of removed stats.
median_bestcase = np.median(removed_bestcase["removed"])
median_worstcase = np.median(removed_worstcase["removed"])
print(f"Median number of removed sidecars (bestcase): {median_bestcase}")
print(f"Median number of removed sidecars (worstcase): {median_worstcase}")

median_bestcase = np.median(removed_bestcase["removedHotspots"])
median_worstcase = np.median(removed_worstcase["removedHotspots"])
print(f"Median number of removed hotspots (bestcase): {median_bestcase}")
print(f"Median number of removed hotspots (worstcase): {median_worstcase}")

fig, ax = plt.subplots(figsize=(4, 3))

data = [removed_bestcase, removed_worstcase]
for i in range(len(data)):
    stats = data[i]
    ax.boxplot(
        [stats["removed"], stats["removedHotspots"]],
        positions=[i * 0.5 + j * 1.25 + 0.5 for j in range(2)],
        widths=0.4,
        showfliers=False,
        patch_artist=True,
        boxprops=dict(
            facecolor=colors[i], color=edge_colors[i], hatch=hatches[i], linewidth=2.5
        ),
        capprops=dict(color=edge_colors[i], linewidth=2.5),
        whiskerprops=dict(color=edge_colors[i], linewidth=2.5),
        flierprops=dict(color=edge_colors[i], markeredgecolor=edge_colors[i]),
        medianprops=dict(color=edge_colors[i], linewidth=2.5),
    )

ax.set_xticks([0.75, 2])
ax.set_xticklabels(["All\nservices", "Hotspot\nservices"], fontsize=24)
ax.set_yticks([0, 0.25, 0.5, 0.75, 1])
ax.set_yticklabels(["0", "", "0.5", "", "1"], fontsize=22)
ax.set_ylabel("Fraction", fontsize=24)

fig.legend(
    handles=[label1, label2],
    loc="center",
    ncol=2,
    frameon=False,
    fontsize=24,
    bbox_to_anchor=(0.5, 0.92),
    bbox_transform=fig.transFigure,
)

plt.tight_layout()
plt.subplots_adjust(left=0.22, right=0.98, top=0.8, bottom=0.25)

plt.savefig("control-plane.pdf")
plt.savefig("control-plane.png")
