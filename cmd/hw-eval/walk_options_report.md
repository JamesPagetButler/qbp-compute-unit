# Walk Phase Hardware Options Comparison

This report compares the four evaluated hardware paths for the QBP Compute Unit, factoring in your existing inventory (1 owned RX 9070 XT).

| Mode | Target CPU | Est. Cost | System TDP | Est. Throughput (QMULs/sec) | Energy Efficiency (Joules/QMUL) | Cost Efficiency (USD per Billion QMULs/s) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **bruteforce** | AMD Ryzen Threadripper PRO 9000 WX-Series | $9144.00 | 1082W | 29.40 Billion | 3.68e-08 J/Op | $311.02 |
| **efficiency** | AMD Ryzen 9 9900X | $1185.00 | 467W | 5.51 Billion | 8.47e-08 J/Op | $214.97 |
| **riscv-beast** | SiFive Performance P870 64-Core (RV64GCV) | $3237.00 | 502W | 14.70 Billion | 3.41e-08 J/Op | $220.20 |
| **riscv-optimized** | SiFive Intelligence X390 16-Core (RV64GCV) | $937.00 | 87W | 2.94 Billion | 2.96e-08 J/Op | $318.71 |

## Key Takeaways
- **Energy Efficiency (Joules/QMUL):** Lower is better. This measures how much power is required for a single mathematical operation.
- **Cost Efficiency:** Lower is better. This measures how much you pay (in hardware CapEx) for every Billion QMULs per second of capacity.
