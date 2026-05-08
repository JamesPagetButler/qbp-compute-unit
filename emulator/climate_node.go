package emulator

import (
	"math"
)

// ClimateNode represents a single cell in the QBP-native Earth System Model.
type ClimateNode struct {
	Pos   QWord
	Vel   QWord
	State QWord
	Spin  QWord
}

func (c *CPU) NewClimateNode() ClimateNode {
	prec := c.GB.Precision()
	return ClimateNode{
		Pos: NewQWord(prec), Vel: NewQWord(prec), State: NewQWord(prec), Spin: NewQWord(prec),
	}
}

// PopulateSphere initializes Q-Mem with ECEF coordinates.
// X = Prime Meridian, Y = 90E, Z = North Pole.
func (c *CPU) PopulateSphere(res int) {
	prec := c.GB.Precision()
	numNodes := res * (res * 2)
	c.Memory = make([]QWord, numNodes*4)

	idx := 0
	for i := 0; i < res; i++ {
		lat := -90.0 + (float64(i)/float64(res-1))*180.0
		phi := lat * math.Pi / 180.0
		for j := 0; j < res*2; j++ {
			lon := -180.0 + (float64(j)/float64(res*2))*360.0
			theta := lon * math.Pi / 180.0

			// AUTHORITATIVE ECEF:
			x := math.Cos(phi) * math.Cos(theta)
			y := math.Cos(phi) * math.Sin(theta)
			z := math.Sin(phi)

			base := idx * 4
			c.Memory[base] = NewQWord(prec)
			c.Memory[base].X.SetFloat64(x)
			c.Memory[base].Y.SetFloat64(y)
			c.Memory[base].Z.SetFloat64(z)

			c.Memory[base+1] = NewQWord(prec)
			c.Memory[base+2] = NewQWord(prec)
			c.Memory[base+3] = NewQWord(prec)
			idx++
		}
	}
}

func (c *CPU) InjectDenseNodes(targetX, targetY, targetZ float64, radius float64, count int) {
	prec := c.GB.Precision()
	newNodes := make([]QWord, count*4)
	for i := 0; i < count; i++ {
		offset := radius * (0.5 + 0.5*math.Sin(float64(i)))
		rx := targetX + (math.Cos(float64(i*7)) * offset)
		ry := targetY + (math.Sin(float64(i*3)) * offset)
		rz := targetZ + (math.Cos(float64(i*5)) * offset)
		mag := math.Sqrt(rx*rx + ry*ry + rz*rz)
		rx, ry, rz = rx/mag, ry/mag, rz/mag
		base := i * 4
		newNodes[base] = NewQWord(prec)
		newNodes[base].X.SetFloat64(rx)
		newNodes[base].Y.SetFloat64(ry)
		newNodes[base].Z.SetFloat64(rz)
		newNodes[base+1] = NewQWord(prec)
		newNodes[base+2] = NewQWord(prec)
		newNodes[base+3] = NewQWord(prec)
	}
	c.Memory = append(c.Memory, newNodes...)
}

func (c *CPU) GetClimateNode(nodeIdx int) ClimateNode {
	base := nodeIdx * 4
	return ClimateNode{Pos: c.Memory[base], Vel: c.Memory[base+1], State: c.Memory[base+2], Spin: c.Memory[base+3]}
}
