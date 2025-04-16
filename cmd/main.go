package main

import (
	// "flag"
	"fmt"
	"log"
	"os"

	_ "image/png"

	"github.com/Kagamiin/pixcrumb/cmd/comp"
	"github.com/Kagamiin/pixcrumb/cmd/imgtools"
)

func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		log.Fatal("error: an input file must be specified")
	}

	for _, filename := range os.Args[1:] {
		fmt.Println("#======================================================================#")
		fmt.Printf("| Test: %-63s|\n", filename)
		fmt.Println("#======================================================================#")

		img, err := imgtools.LoadImage(filename)
		if err != nil {
			log.Printf("\nERROR: Could not load image file '%s': %s\n\n", filename, err.Error())
			continue
		}

		planarImg, err := imgtools.NewPlanarImage(img)
		if err != nil {
			log.Printf("ERROR: %s", err.Error())
			continue
		}
		bitplanes := planarImg.GetBitplanes()

		for _, bp := range bitplanes {
			bp.DeltaEncode()
		}

		crumbImage := imgtools.ImagePlanarToCrumb(planarImg)
		crumbPlanes := crumbImage.GetPlanes()

		encoders := []comp.PixCrumbEncoder{
			comp.NewPixCrumbRLE(),
		}

		var totalSizeRaw, totalSizeComp uint64
		for _, pixcrumbEncoder := range encoders {
			totalSizeRaw = 0
			totalSizeComp = 0
			fmt.Printf("\nUsing method %s:\n", pixcrumbEncoder.GetName())
			for i, crp := range crumbPlanes {
				rawSize := bitplanes[i].GetTotalSize()
				comp, err := pixcrumbEncoder.Compress(&crp)
				if err != nil {
					log.Printf("error while encoding BP%d: %s\n", i, err.Error())
					continue
				}
				compSize := comp.GetTotalSize()
				fmt.Printf("BP%d raw size: %d bytes, compressed to %d bytes (ratio: %.03f)\n", i, rawSize, compSize, float64(compSize)/float64(rawSize))
				totalSizeRaw += rawSize
				totalSizeComp += compSize
			}
			fmt.Printf("Total: raw size %d bytes, compressed to %d bytes (ratio: %.03f)\n\n", totalSizeRaw, totalSizeComp, float64(totalSizeComp)/float64(totalSizeRaw))
		}
	}
}

func handle(err error) {
	if err != nil {
		log.Fatal("error: ", err)
	}
}
