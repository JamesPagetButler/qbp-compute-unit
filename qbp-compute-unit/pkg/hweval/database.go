package hweval

type PartCategory string

const (
	CPU   PartCategory = "CPU"
	GPU   PartCategory = "GPU"
	RAM   PartCategory = "RAM"
	MOBO  PartCategory = "Motherboard"
	PSU   PartCategory = "PowerSupply"
	DRIVE PartCategory = "Storage"
)

type Component struct {
	ID          string
	Category    PartCategory
	Name        string
	Cost        float64 // USD
	TDPWatts    float64
	AVX512      bool // Or 512-bit Vector extension for RISC-V
	Cores       int
	ClockGHz    float64
	MemChannels int
	Notes       string
}

// Catalog represents the 2026 available hardware parts
var Catalog = []Component{
	// x86 CPUs
	{ID: "cpu-tr-9000wx", Category: CPU, Name: "AMD Ryzen Threadripper PRO 9000 WX-Series", Cost: 4999.00, TDPWatts: 350.0, AVX512: true, Cores: 64, ClockGHz: 5.0, MemChannels: 8, Notes: "Massive QW128 parallelism"},
	{ID: "cpu-r9-9900x", Category: CPU, Name: "AMD Ryzen 9 9900X", Cost: 499.00, TDPWatts: 120.0, AVX512: true, Cores: 12, ClockGHz: 5.0, MemChannels: 2, Notes: "Excellent AVX-512 cost/power ratio"},

	// RISC-V CPUs
	{ID: "cpu-sifive-p870", Category: CPU, Name: "SiFive Performance P870 64-Core (RV64GCV)", Cost: 1500.00, TDPWatts: 120.0, AVX512: true, Cores: 64, ClockGHz: 2.5, MemChannels: 4, Notes: "RISC-V Beast Mode Server"},
	{ID: "cpu-sifive-x390", Category: CPU, Name: "SiFive Intelligence X390 16-Core (RV64GCV)", Cost: 350.00, TDPWatts: 35.0, AVX512: true, Cores: 16, ClockGHz: 2.0, MemChannels: 2, Notes: "Optimized Edge Node"},

	// GPUs
	{ID: "gpu-rx9070xt", Category: GPU, Name: "AMD Radeon RX 9070 XT 32GB", Cost: 1199.00, TDPWatts: 300.0, AVX512: false, Cores: 0, ClockGHz: 0.0, MemChannels: 0, Notes: "ROCm native Walk phase accelerator"},
	{ID: "gpu-integrated", Category: GPU, Name: "Integrated RDNA/Vector Graphics", Cost: 0.00, TDPWatts: 15.0, AVX512: false, Cores: 0, ClockGHz: 0.0, MemChannels: 0, Notes: "Included with CPU"},

	// Motherboards
	{ID: "mobo-wrx90", Category: MOBO, Name: "WRX90 E-ATX Workstation", Cost: 1299.00, TDPWatts: 80.0, AVX512: false, Cores: 0, ClockGHz: 0.0, MemChannels: 8, Notes: "Required for Threadripper"},
	{ID: "mobo-x870", Category: MOBO, Name: "X870 Mini-ITX", Cost: 249.00, TDPWatts: 30.0, AVX512: false, Cores: 0, ClockGHz: 0.0, MemChannels: 2, Notes: "Efficient edge node board"},
	{ID: "mobo-rv-server", Category: MOBO, Name: "RISC-V E-ATX Server Board", Cost: 800.00, TDPWatts: 50.0, AVX512: false, Cores: 0, ClockGHz: 0.0, MemChannels: 4, Notes: "Server board for P870"},
	{ID: "mobo-rv-itx", Category: MOBO, Name: "RISC-V Mini-ITX Edge Board", Cost: 150.00, TDPWatts: 20.0, AVX512: false, Cores: 0, ClockGHz: 0.0, MemChannels: 2, Notes: "Edge board for X390"},

	// RAM
	{ID: "ram-256gb-ddr5", Category: RAM, Name: "256GB (8x32GB) DDR5-7200 RDIMM", Cost: 899.00, TDPWatts: 40.0, AVX512: false, Cores: 0, ClockGHz: 0.0, MemChannels: 8, Notes: "Feeds 8-channel TR PRO"},
	{ID: "ram-128gb-ddr5", Category: RAM, Name: "128GB (4x32GB) DDR5-6000", Cost: 389.00, TDPWatts: 20.0, AVX512: false, Cores: 0, ClockGHz: 0.0, MemChannels: 4, Notes: "Standard for Server"},
	{ID: "ram-64gb-ddr5", Category: RAM, Name: "64GB (2x32GB) DDR5-6000", Cost: 189.00, TDPWatts: 10.0, AVX512: false, Cores: 0, ClockGHz: 0.0, MemChannels: 2, Notes: "Standard edge node memory"},

	// Storage
	{ID: "nvme-4tb-gen5", Category: DRIVE, Name: "4TB PCIe Gen 5 NVMe SSD", Cost: 399.00, TDPWatts: 12.0, AVX512: false, Cores: 0, ClockGHz: 0.0, MemChannels: 0, Notes: ""},
	{ID: "nvme-2tb-gen4", Category: DRIVE, Name: "2TB PCIe Gen 4 NVMe SSD", Cost: 129.00, TDPWatts: 7.0, AVX512: false, Cores: 0, ClockGHz: 0.0, MemChannels: 0, Notes: ""},

	// Power Supplies
	{ID: "psu-1600w", Category: PSU, Name: "1600W 80+ Titanium ATX 3.1", Cost: 349.00, TDPWatts: 0.0, AVX512: false, Cores: 0, ClockGHz: 0.0, MemChannels: 0, Notes: ""},
	{ID: "psu-850w", Category: PSU, Name: "850W 80+ Platinum", Cost: 149.00, TDPWatts: 0.0, AVX512: false, Cores: 0, ClockGHz: 0.0, MemChannels: 0, Notes: ""},
	{ID: "psu-500w", Category: PSU, Name: "500W 80+ Platinum SFX", Cost: 119.00, TDPWatts: 0.0, AVX512: false, Cores: 0, ClockGHz: 0.0, MemChannels: 0, Notes: ""},
}
