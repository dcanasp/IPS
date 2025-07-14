import matplotlib.pyplot as plt
import numpy as np
import json
from matplotlib.widgets import Slider

# Simulated features (examples): [request_rate, error_rate, unique_paths, header_var]
# Format: [feature1, feature2, feature3, feature4]
sample_data = np.array([
    [5, 0.1, 3, 1],   # Real user
    [80, 0.3, 12, 7], # DDoS
    [40, 0.9, 1, 2],  # High error
    [10, 0.0, 10, 0], # Explorer
    [15, 0.2, 3, 8],  # Header spoof
])

labels = ["Real", "DDoS", "Fuzzer", "Explorer", "Spoof"]

# Default weights
weights = np.array([0.1, 5.0, 0.4, 0.3])  # W1, W2, W3, W4

def softmax_score(features, weights):
    logits = np.dot(features, weights)
    return 1 / (1 + np.exp(-logits))  # logistic sigmoid

# --- Plotting ---
fig, ax = plt.subplots()
plt.subplots_adjust(left=0.3, bottom=0.4)
bars = ax.bar(labels, [softmax_score(row, weights) for row in sample_data])
ax.set_ylim(0, 1)
ax.set_ylabel("Attack Probability (Softmax)")

# --- Sliders for weights ---
slider_ax = [plt.axes([0.3, 0.3 - i * 0.05, 0.6, 0.03]) for i in range(4)]
sliders = [Slider(slider_ax[i], f'W{i+1}', 0.0, 10.0, valinit=weights[i]) for i in range(4)]

def update(val):
    new_weights = np.array([s.val for s in sliders])
    new_scores = [softmax_score(row, new_weights) for row in sample_data]
    for bar, score in zip(bars, new_scores):
        bar.set_height(score)
    fig.canvas.draw_idle()

for s in sliders:
    s.on_changed(update)

plt.show()
