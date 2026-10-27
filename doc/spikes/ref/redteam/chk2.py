import numpy as np, flowlib as F
rng=np.random.default_rng(2); r=0
for _ in range(500):
    s=F.random_states(1,rng)[0]; a=s[:8].copy();a[0]=0;c=s[8:].copy();c[0]=0
    r=max(r,abs(F.potential(s)-4*((a@a)*(c@c)-(a@c)**2)))
print("V - 4*GramDet(a,c) max:",r)
# generic SO(7) rotation (NOT a G2 automorphism) on imaginary parts of both halves
M=np.linalg.qr(rng.normal(size=(7,7)))[0]
R=np.eye(8);R[1:,1:]=M
s=F.random_states(1,rng)[0]; t=s.copy(); t[:8]=R@s[:8]; t[8:]=R@s[8:]
print("V(s), V(SO7-rotated s):",F.potential(s),F.potential(t))
