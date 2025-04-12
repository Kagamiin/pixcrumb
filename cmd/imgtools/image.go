package imgtools

import (
	"fmt"
	"image"
	"image/color"
	"os"
)

func LoadImage(filename string) (image.PalettedImage, error) {
	reader, err := os.Open(filename)
	defer reader.Close()
	if err != nil {
		return nil, err
	}
	im, _, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}

	if pim, ok := im.(image.PalettedImage); ok {
		if _, ok = pim.ColorModel().(color.Palette); !ok {
			return nil, fmt.Errorf("input image '%s' is not paletted", filename)
		}
	} else {
		return nil, fmt.Errorf("input image '%s' is not paletted", filename)
	}

	pim := im.(image.PalettedImage)
	pal := pim.ColorModel().(color.Palette)
	numColors := len(pal)
	if numColors < 2 {
		return nil, fmt.Errorf("input image '%s' only has %d color in palette (needs to have at least 2)", filename, len(pal))
	}
	if (numColors & (numColors - 1)) != 0 {
		return nil, fmt.Errorf("input image '%s' has %d colors in palette (number of colors needs to be a power of 2)", filename, len(pal))
	}

	return pim, nil
}
