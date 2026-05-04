# BMA Phase 0: System Verification, Updates & Optimization

## 0.0 — Version Inventory

Run all of these and record the output before changing anything.
This is the "before" snapshot.

```bash
# System
cat /etc/os-release
uname -r
sudo dmidecode -s bios-version
sudo dmidecode -s bios-release-date

# GPU
rocm-smi --version
rocm-smi  # full status
vulkaninfo --summary 2>/dev/null || echo "vulkaninfo not installed"
sudo cat /sys/class/drm/card*/device/vbios_version

# Storage
sudo smartctl -a /dev/sda  # adjust device path as needed
lsblk -f  # filesystem types and mount points

# CPU/Thermal
sensors
cat /proc/cpuinfo | head -30
cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor

# Memory
free -h
cat /proc/meminfo | grep -E "MemTotal|SwapTotal|Hugepages"
cat /proc/sys/vm/swappiness

# Container runtime
podman --version
podman info | grep -E "rootless|cgroup"

# Development
go version 2>/dev/null || echo "Go not installed"

# Network
tailscale version 2>/dev/null || echo "Tailscale not installed"
```

## 0.1 — Update Assessment

For each component, determine: current version, latest available,
minimum required for BMA, and whether an update is needed.

| Component | Minimum for BMA | Check | Update Path |
|-----------|----------------|-------|-------------|
| **BIOS** | Latest available for Crosshair V Formula-Z | Compare dmidecode output to ASUS support page | USB flash from BIOS. Download from asus.com. Risk: low but non-reversible. |
| **Kernel** | 6.12+ (RDNA 4 amdgpu support, fan speed, thermal) | `uname -r` | Pop!_OS may offer linux-edge. Or install mainline kernel. Risk: medium — test before committing. |
| **ROCm** | 6.4.2+ (RDNA 4 official support) | `apt list --installed \| grep rocm` | Add AMD's official repo. `amdgpu-install --usecase=rocm`. Risk: low if following AMD docs. |
| **Mesa/Vulkan** | Mesa 24.1+ (RADV RDNA 4 support) | `vulkaninfo --summary` | PPA or build from source if Pop!_OS lags. Needed for Kaiju in late Crawl. |
| **Podman** | 4.0+ (rootless device passthrough) | `podman --version` | Pop!_OS repo or Podman upstream repo. Risk: low. |
| **Go** | 1.22+ (latest stable) | `go version` | golang.org/dl or distrobox with Go image. |
| **Tailscale** | Latest stable | `tailscale version` | curl -fsSL https://tailscale.com/install.sh | sh |
| **GPU firmware** | Check PowerColor for 9070 XT updates | Compare vbios_version to PowerColor support | Usually updated via amdgpu driver. Risk: low. |
| **SSD firmware** | Check Samsung for 840 updates | `smartctl -i` firmware version | Samsung Magician (may need temp Windows boot) or linux firmware tool. Likely no updates for this age. |

## 0.2 — Apply Updates (in order)

Order matters. BIOS first (requires reboot), kernel second
(requires reboot), then userspace packages (no reboot).

1. **BIOS** — Only if ASUS has a newer version. Flash via USB.
   Reboot. Verify IOMMU enabled in BIOS settings (IOMMU = Enabled).
2. **Kernel** — Install newer kernel if needed. Reboot.
   Verify: `uname -r` shows new version. `dmesg | grep amdgpu`
   shows RDNA 4 recognized.
3. **ROCm** — Install/upgrade. Verify: `rocm-smi` shows 9070 XT
   with temperature, VRAM, clocks.
4. **Podman** — Install/upgrade. Verify: `podman --version`.
5. **Go** — Install in distrobox or host. Verify: `go version`.
6. **Tailscale** — Install on host and Android phone.
   Verify: `tailscale status` shows both devices.
7. **Mesa/Vulkan** — Upgrade if needed for Kaiju later. Not
   blocking for early Crawl.

## 0.3 — Sensor Verification (inside container)

Only run this AFTER updates are applied.

```bash
# GPU visible and reporting
podman run --rm \
  --device /dev/kfd --device /dev/dri \
  --group-add keep-groups \
  rocm/dev-ubuntu-24.04:latest rocm-smi

# GPU hwmon temperature
podman run --rm \
  --device /dev/kfd --device /dev/dri \
  --group-add keep-groups \
  rocm/dev-ubuntu-24.04:latest \
  sh -c "find /sys/class/hwmon -name 'temp*_input' -exec cat {} \;"

# CPU temp (may need host hwmon bind mount)
podman run --rm \
  -v /sys/class/hwmon:/sys/class/hwmon:ro \
  rocm/dev-ubuntu-24.04:latest \
  sh -c "find /sys/class/hwmon -name 'temp*_input' -exec sh -c 'echo {}; cat {}' \;"

# Cgroup limits
podman run --rm --memory 20g --cpus 6 \
  rocm/dev-ubuntu-24.04:latest \
  sh -c "cat /sys/fs/cgroup/memory.max && cat /sys/fs/cgroup/cpu.max"

# Disk I/O through bind mount
podman run --rm -v /tmp:/tmp:Z \
  rocm/dev-ubuntu-24.04:latest \
  dd if=/dev/zero of=/tmp/bma_iotest bs=1M count=256 oflag=direct 2>&1

# PCIe bandwidth
podman run --rm \
  --device /dev/kfd --device /dev/dri \
  --group-add keep-groups \
  rocm/dev-ubuntu-24.04:latest \
  rocm-bandwidth-test 2>&1 | head -40
```

## 0.4 — System Optimization

Apply AFTER verification passes. Each change includes rationale,
expected impact, and revert instructions.

### I/O Scheduler
```bash
# Check current
cat /sys/block/sda/queue/scheduler
# Samsung 840 on SATA: 'mq-deadline' is optimal for mixed r/w
echo mq-deadline | sudo tee /sys/block/sda/queue/scheduler
# Persist: add to /etc/udev/rules.d/60-ioscheduler.rules
# ACTION=="add|change", KERNEL=="sd*", ATTR{queue/scheduler}="mq-deadline"
# Revert: echo none | sudo tee /sys/block/sda/queue/scheduler
```

### Swappiness
```bash
# Check current
cat /proc/sys/vm/swappiness
# Default is 60. For BMA: lower to 10.
# Keeps hypergraph index in RAM under pressure.
# Host may swap before BMA's container memory is reclaimed.
sudo sysctl vm.swappiness=10
# Persist: echo "vm.swappiness=10" | sudo tee -a /etc/sysctl.d/99-bma.conf
# Revert: sudo sysctl vm.swappiness=60
```

### CPU Governor
```bash
# Check current
cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor
# FX-8350 default: ondemand. For BMA inference: performance.
# Keeps clocks high during inference. Higher power draw.
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor
# Or selectively: only during BMA operation, revert to ondemand after.
# Persist: install cpufrequtils, set GOVERNOR="performance" in /etc/default/cpufrequtils
# Revert: echo ondemand | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor
```

### Transparent Huge Pages
```bash
# Check current
cat /sys/kernel/mm/transparent_hugepage/enabled
# For Go runtime with large heap (hypergraph index): 'madvise' is safest.
# 'always' can cause latency spikes from compaction.
echo madvise | sudo tee /sys/kernel/mm/transparent_hugepage/enabled
# Revert: echo always | sudo tee /sys/kernel/mm/transparent_hugepage/enabled
```

### GPU Power Profile
```bash
# Inside container with GPU access:
# Check current
cat /sys/class/drm/card0/device/power_dpm_force_performance_level
# For sustained inference: 'high' gives consistent clocks.
# Default 'auto' may downclock during pauses between inference calls.
echo high | sudo tee /sys/class/drm/card0/device/power_dpm_force_performance_level
# WARNING: increases power draw and temperature. Monitor with rocm-smi.
# Revert: echo auto | sudo tee /sys/class/drm/card0/device/power_dpm_force_performance_level
```

### ROCm Environment
```bash
# For RDNA 4 consumer GPU, may need:
export HSA_OVERRIDE_GFX_VERSION=12.0.0  # if rocm doesn't recognize 9070 XT natively
# Check if needed: run rocminfo inside container. If GPU shows up, not needed.
# If needed, add to container env: podman run -e HSA_OVERRIDE_GFX_VERSION=12.0.0 ...
```

### Filesystem Considerations
```bash
# Check current filesystem on data partition
df -T /home
# If ext4: consider creating a btrfs subvolume for bma-data
# Benefits: snapshots (podman commit equivalent for data), compression
# Risk: btrfs on SATA SSD is stable but adds write amplification
# Alternative: stay on ext4, use tar for snapshots (simpler, no risk)
# Decision: evaluate after measuring write patterns in early Crawl
```

## 0.5 — Post-Optimization Re-Verification

Re-run the sensor verification (0.3) after all optimizations.
Compare to baseline. Document the delta.

```bash
# Quick comparison script
echo "=== I/O Scheduler ===" && cat /sys/block/sda/queue/scheduler
echo "=== Swappiness ===" && cat /proc/sys/vm/swappiness
echo "=== CPU Governor ===" && cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor
echo "=== THP ===" && cat /sys/kernel/mm/transparent_hugepage/enabled
echo "=== GPU Power ===" && cat /sys/class/drm/card0/device/power_dpm_force_performance_level
```

Store this output alongside the Phase 0.0 inventory as the
"after" snapshot. Both go into the BMA repo as reference documents.

## 0.6 — Phase 0 Gate

All of the following must be true before Phase 1 (BMA-PROBE) begins:

- [ ] Pop!_OS running with kernel supporting RDNA 4 amdgpu
- [ ] ROCm 6.4.2+ installed and rocm-smi shows 9070 XT
- [ ] Podman installed, rootless mode working
- [ ] GPU accessible inside Podman container (rocm-smi works)
- [ ] GPU temperature readable inside container
- [ ] CPU temperature readable (host or bind-mounted)
- [ ] Cgroup limits enforced and visible inside container
- [ ] SATA SSD healthy (SMART OK, endurance > 50%)
- [ ] Disk I/O baseline recorded (host and container)
- [ ] PCIe bandwidth measured (~8 GB/s expected)
- [ ] Tailscale installed on host and phone, both connected
- [ ] Go installed (distrobox or host)
- [ ] All optimizations applied and verified
- [ ] Before/after snapshots committed to repo

## 0.7 — Available Resources

### API Accounts
```bash
# Claude Max API ($100/month)
# Verify API key works
curl -s https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "content-type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{"model":"claude-sonnet-4-20250514","max_tokens":10,"messages":[{"role":"user","content":"ping"}]}' | head -1
# Expected: valid JSON response. Budget: ~$100/month.
# At Code Mode rates ($0.028/complex exchange): ~3,500 exchanges/month.

# Gemini Ultra
# Verify Gemini API access
curl -s "https://generativelanguage.googleapis.com/v1beta/models?key=$GEMINI_API_KEY" | head -5
# Expected: model list including gemini-2.5-pro or similar.
# Key advantage: 1M+ token context window for rich domain context.
```

### Quantum Randomness: CURBy
```bash
# Colorado University Randomness Beacon
# https://random.colorado.edu/
# NIST + University of Colorado Boulder
# 512 bits of quantum-certified true randomness per pulse
# Verifiable via Twine protocol (blockchain-style audit trail)
curl -s https://random.colorado.edu/api/latest 2>/dev/null || echo "Check CURBy API endpoint"
# Expected: JSON with random bits, timestamp, verification hash.
# Use: seed BMA's random operations (sleep consolidation ordering,
# epistemic map probing, seed generation verification).
# True quantum randomness eliminates subtle biases that pseudo-random
# sources introduce in consolidation decisions.
```

### Phase 0.7 Gate
- [ ] Anthropic API key valid. Budget confirmed.
- [ ] Gemini API key valid. Ultra tier confirmed.
- [ ] CURBy reachable. Random pulse retrievable.
- [ ] All three resources documented in BMA config with endpoints and budget limits.
