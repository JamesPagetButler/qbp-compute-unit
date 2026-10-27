# Reference integrators — benchmark_multidim_cu_on_688.py (sandbox), verbatim
# Source: web Gemini, 2026-10-16. Provenance record for spike A2 (doc/spikes/mdcu-xqbp-v2-spike.md).
# The Go ports in emulator/mdcu/workloads/ruleflow.go must match these line-for-line.
import numpy as np

# 1. Baseline & CD-MAC (16D state s on S^14, with s[0] == 0)
def rk4_step_16d(s, h, ruleField_fn):
    k1 = ruleField_fn(s)
    s2 = s + 0.5 * h * k1; s2[..., 0] = 0.0; s2 /= np.linalg.norm(s2, axis=-1, keepdims=True)
    k2 = ruleField_fn(s2)
    s3 = s + 0.5 * h * k2; s3[..., 0] = 0.0; s3 /= np.linalg.norm(s3, axis=-1, keepdims=True)
    k3 = ruleField_fn(s3)
    s4 = s + h * k3;       s4[..., 0] = 0.0; s4 /= np.linalg.norm(s4, axis=-1, keepdims=True)
    k4 = ruleField_fn(s4)
    t  = s + (h / 6.0) * (k1 + 2.0*k2 + 2.0*k3 + k4)
    t[..., 0] = 0.0
    return t / np.linalg.norm(t, axis=-1, keepdims=True)

# 2. Leaf-Foliated (5D state q = [a1, a2, c1, c2, b0] on S^4)
def rk4_step_leaf(q, h):
    k1 = leaf_ruleField(q)
    q2 = q + 0.5 * h * k1; q2 /= np.linalg.norm(q2, axis=-1, keepdims=True)
    k2 = leaf_ruleField(q2)
    q3 = q + 0.5 * h * k2; q3 /= np.linalg.norm(q3, axis=-1, keepdims=True)
    k3 = leaf_ruleField(q3)
    q4 = q + h * k3;       q4 /= np.linalg.norm(q4, axis=-1, keepdims=True)
    k4 = leaf_ruleField(q4)
    qn = q + (h / 6.0) * (k1 + 2.0*k2 + 2.0*k3 + k4)
    return qn / np.linalg.norm(qn, axis=-1, keepdims=True)

# 3. Leaf frame + field (extract_leaf_frame / leaf_ruleField, lines 102-154)
def extract_leaf_frame(s0):
    a = s0[..., :8].copy(); a[..., 0] = 0.0
    c = s0[..., 8:].copy(); b0_val = c[..., 0:1].copy(); c[..., 0] = 0.0
    a1 = np.linalg.norm(a, axis=-1, keepdims=True)
    u1 = a / np.maximum(a1, 1e-30)
    c1 = np.sum(c * u1, axis=-1, keepdims=True)
    c_perp = c - c1 * u1
    c2 = np.linalg.norm(c_perp, axis=-1, keepdims=True)
    u2 = c_perp / np.maximum(c2, 1e-30)
    a2 = np.zeros_like(a1)
    q = np.concatenate([a1, a2, c1, c2, b0_val], axis=-1)
    return q, u1, u2

def leaf_ruleField(q):
    a1, a2, c1, c2, b0_val = q[..., 0:1], q[..., 1:2], q[..., 2:3], q[..., 3:4], q[..., 4:5]
    det = a1 * c2 - a2 * c1
    ga1 = 8.0 * det * c2
    ga2 = -8.0 * det * c1
    gc1 = -8.0 * det * a2
    gc2 = 8.0 * det * a1
    gb0 = np.zeros_like(b0_val)
    g = np.concatenate([ga1, ga2, gc1, gc2, gb0], axis=-1)
    rad = np.sum(g * q, axis=-1, keepdims=True) * q
    return -(g - rad)
