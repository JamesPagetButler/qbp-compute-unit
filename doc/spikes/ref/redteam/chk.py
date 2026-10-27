import numpy as np, flowlib as F
rng=np.random.default_rng(1)
worst=0; worstplane=0
for _ in range(200):
    s=F.random_states(1,rng)[0] if F.random_states(1,rng).ndim>1 else F.random_states(1,rng)
    lo,hi=s[:8],s[8:]
    a=lo.copy(); a[0]=0; c=hi.copy(); c[0]=0
    g=F.gradV_exact(s)
    Na,Nc,b=a@a,c@c,a@c
    pl=8*Nc*a-8*b*c; ph=8*Na*c-8*b*a
    worst=max(worst,np.abs(g[:8]-pl).max(),np.abs(g[8:]-ph).max())
print("Thm3 max residual vs gradV_exact:",worst)
# leaf invariance: run flowlib.step and track distance of imag parts from initial plane span{a0,c0}
s=F.random_states(1,rng); s=s[0] if s.ndim>1 else s
a0=s[:8].copy();a0[0]=0;c0=s[8:].copy();c0[0]=0
Q,_=np.linalg.qr(np.stack([a0,c0],1))
mx=0
for t in range(2000):
    s=F.step(s,0.02)
    for v in (s[:8].copy(),s[8:].copy()):
        v[0]=0; mx=max(mx,np.linalg.norm(v-Q@(Q.T@v)))
print("max out-of-plane drift over 2000 flowlib steps:",mx, " final V:",F.potential(s))
