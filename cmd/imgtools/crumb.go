package imgtools

import (
	"image/color"
	"math"
)

// Crumb represents a 2x2 pixel region in a bitplane.
// The bits are organized as such:
// +---+---+
// | 3 | 2 |
// +---+---+
// | 1 | 0 |
// +---+---+
type Crumb uint8

type CrumbPlane struct {
	crumbs [][]Crumb
	width  uint64
	height uint64
}

func (c CrumbPlane) GetWidthPx() uint64 {
	return c.width
}

func (c CrumbPlane) GetWidthCrumbs() uint64 {
	return uint64(math.Ceil(float64(c.width) / 2))
}

func (c CrumbPlane) GetWidthBpBytes() uint64 {
	return uint64(math.Ceil(float64(c.width) / 8))
}

func (c CrumbPlane) GetHeightPx() uint64 {
	return c.height
}

func (c CrumbPlane) GetHeightCrumbs() uint64 {
	return uint64(math.Ceil(float64(c.height) / 2))
}

func (c CrumbPlane) GetCrumbs() [][]Crumb {
	return c.crumbs
}

type CrumbImage struct {
	planes  []CrumbPlane
	palette color.Palette
	width   uint64
	height  uint64
}

func (c CrumbImage) GetPlanes() []CrumbPlane {
	return c.planes
}

func ImagePlanarToCrumb(pi *PlanarImage) *CrumbImage {
	result := CrumbImage{
		planes:  make([]CrumbPlane, 0),
		palette: pi.palette,
		width:   pi.width,
		height:  pi.height,
	}
	for _, bp := range pi.planes {
		crp := BitplaneToCrumbPlane(&bp)
		result.planes = append(result.planes, *crp)
	}
	return &result
}

func BitplaneToCrumbPlane(bp *Bitplane) *CrumbPlane {
	crumbsH := int(math.Ceil(float64(bp.height) / 2))
	crumbsW := int(math.Ceil(float64(bp.width) / 2))

	result := CrumbPlane{
		crumbs: make([][]Crumb, crumbsH),
		width:  bp.width,
		height: bp.height,
	}

	var bpRowPair [2][]byte
	for i, _ := range result.crumbs {
		bpRowPair[0] = bp.data[i*2]
		if i*2+1 < int(bp.height) {
			bpRowPair[1] = bp.data[i*2+1]
		} else {
			bpRowPair[1] = make([]byte, len(bp.data[i*2]))
		}
		result.crumbs[i] = bpRowPairToCrumbRow(bpRowPair, uint64(crumbsW))
	}

	return &result
}

func bpRowPairToCrumbRow(bpRowPair [2][]byte, crumbsW uint64) []Crumb {
	result := make([]Crumb, crumbsW)
	for i, b := range bpRowPair[0] {
		crumbOffs := i * 4
		for ii := 0; ii < 3 && crumbOffs < int(crumbsW); ii++ {
			result[crumbOffs] |= Crumb((b & 0xC0) >> 4)
			crumbOffs++
			b <<= 2
		}
	}
	for i, b := range bpRowPair[1] {
		crumbOffs := i * 4
		for ii := 0; ii < 3 && crumbOffs < int(crumbsW); ii++ {
			result[crumbOffs] |= Crumb((b & 0xC0) >> 6)
			crumbOffs++
			b <<= 2
		}
	}
	return result
}
