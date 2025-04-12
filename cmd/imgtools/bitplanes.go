package imgtools

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

type Bitplane struct {
	data   [][]byte
	width  uint64
	height uint64
}

func (b *Bitplane) GetWidthBpBytes() uint64 {
	return uint64(math.Ceil(float64(b.width) / 8))
}

func (b *Bitplane) GetHeightPx() uint64 {
	return b.height
}

func (b *Bitplane) GetTotalSize() uint64 {
	return b.GetHeightPx() * b.GetWidthBpBytes()
}

func (b *Bitplane) DeltaEncode() {
	deltaBuffer := make([]byte, b.GetWidthBpBytes())
	for i, line := range b.data[1:] {
		for j, v := range line {
			deltaBuffer[j] = b.data[i][j] ^ v
		}
		if i > 0 {
			copy(b.data[i], deltaBuffer)
		}
	}
	b.data[len(b.data)-1] = deltaBuffer
}

type PlanarImage struct {
	planes  []Bitplane
	palette color.Palette
	width   uint64
	height  uint64
}

func (i PlanarImage) GetBitplanes() []Bitplane {
	return i.planes
}

func NewPlanarImage(im image.PalettedImage) (*PlanarImage, error) {
	if im == nil {
		panic("im is nil")
	}
	bounds := im.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y
	numColors := len(im.ColorModel().(color.Palette))
	numBitplanes := int(math.Ceil(math.Log2(float64(numColors))))

	if numBitplanes > 16 {
		return nil, fmt.Errorf("input image has too many colors! (how in the world did you load an image with %d colors, which is more than 65536?)", numColors)
	}

	// Note: despite this being a triply-nested loop, numBitplanes is determined by the log2 of numColors.
	// Realistically, numColors will never be above 256, and in practice a value of 2, 4 or 16 (yielding 1, 2 or 4 bps) is more likely to be expected.
	result := PlanarImage{
		palette: im.ColorModel().(color.Palette),
		width:   uint64(width),
		height:  uint64(height),
	}

	for b := 0; b < numBitplanes; b++ {
		plane := Bitplane{
			data:   make([][]byte, height),
			width:  uint64(width),
			height: uint64(height),
		}
		for i := 0; i < height; i++ {
			plane.data[i] = make([]byte, int(math.Ceil(float64(width)/8)))
			for j := 0; j < int(math.Ceil(float64(width)/8)); j++ {
				for bit := 0; bit < 8; bit++ {
					xpos := (j * 8) + bit
					plane.data[i][j] <<= 1
					bitVal := uint8(0)
					if xpos < width && (im.ColorIndexAt(bounds.Min.X+xpos, bounds.Min.Y+i)&(1<<b)) != 0 {
						bitVal = 1
					}
					plane.data[i][j] |= bitVal
				}
			}
		}
		result.planes = append(result.planes, plane)
	}

	return &result, nil
}
