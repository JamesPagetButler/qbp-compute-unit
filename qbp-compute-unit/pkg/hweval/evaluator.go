package hweval

import (
	"fmt"
)

type BuildMode string

const (
	ModeBruteForce      BuildMode = "bruteforce"
	ModeEfficiency      BuildMode = "efficiency"
	ModeServer          BuildMode = "server"
	ModeRiscvBeast      BuildMode = "riscv-beast"
	ModeRiscvOptimized  BuildMode = "riscv-optimized"
)

type SelectedPart struct {
	Component Component
	Quantity  int
	Cost      float64 // 0 if owned
}

type BuildReport struct {
	Mode         BuildMode
	Parts        []SelectedPart
	TotalCost    float64
	TotalTDP     float64
	CPU          Component // Reference to CPU for performance math
}

func getComponent(id string) Component {
	for _, c := range Catalog {
		if c.ID == id {
			return c
		}
	}
	panic("Unknown component ID: " + id)
}

func Evaluate(mode BuildMode, ownedGPUs int) BuildReport {
	report := BuildReport{
		Mode:  mode,
		Parts: []SelectedPart{},
	}

	addPart := func(id string, qty int, owned int) {
		comp := getComponent(id)
		
		if comp.Category == CPU {
			report.CPU = comp
		}

		chargeableQty := qty - owned
		if chargeableQty < 0 {
			chargeableQty = 0
		}

		cost := comp.Cost * float64(chargeableQty)
		
		report.Parts = append(report.Parts, SelectedPart{
			Component: comp,
			Quantity:  qty,
			Cost:      cost,
		})
		
		report.TotalCost += cost
		report.TotalTDP += comp.TDPWatts * float64(qty)
	}

	switch mode {
	case ModeBruteForce:
		addPart("cpu-tr-9000wx", 1, 0)
		addPart("mobo-wrx90", 1, 0)
		addPart("ram-256gb-ddr5", 1, 0)
		addPart("gpu-rx9070xt", 2, ownedGPUs)
		addPart("nvme-4tb-gen5", 1, 0)
		addPart("psu-1600w", 1, 0)

	case ModeEfficiency:
		addPart("cpu-r9-9900x", 1, 0)
		addPart("mobo-x870", 1, 0)
		addPart("ram-64gb-ddr5", 1, 0)
		if ownedGPUs > 0 {
			addPart("gpu-rx9070xt", 1, ownedGPUs)
		} else {
			addPart("gpu-integrated", 1, 0)
		}
		addPart("nvme-2tb-gen4", 1, 0)
		addPart("psu-500w", 1, 0)

	case ModeRiscvBeast:
		addPart("cpu-sifive-p870", 1, 0)
		addPart("mobo-rv-server", 1, 0)
		addPart("ram-128gb-ddr5", 1, 0)
		if ownedGPUs > 0 {
			addPart("gpu-rx9070xt", 1, ownedGPUs)
		} else {
			addPart("gpu-integrated", 1, 0)
		}
		addPart("nvme-4tb-gen5", 1, 0)
		addPart("psu-850w", 1, 0)

	case ModeRiscvOptimized:
		addPart("cpu-sifive-x390", 1, 0)
		addPart("mobo-rv-itx", 1, 0)
		addPart("ram-64gb-ddr5", 1, 0)
		addPart("gpu-integrated", 1, 0)
		addPart("nvme-2tb-gen4", 1, 0)
		addPart("psu-500w", 1, 0)

	case ModeServer:
		fmt.Println("Server mode currently requires custom ASIC tape-out estimation not supported by off-the-shelf catalog.")
	}

	return report
}
