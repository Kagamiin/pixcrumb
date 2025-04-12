package main

import (
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

	filename := os.Args[1]
	img, err := imgtools.LoadImage(filename)
	if err != nil {
		log.Fatalf("\nERROR: Could not load image file '%s': %s\n\n", filename, err.Error())
	}

	planarImg, err := imgtools.NewPlanarImage(img)
	if err != nil {
		log.Fatalf("ERROR: %s", err.Error())
	}
	bitplanes := planarImg.GetBitplanes()

	for _, bp := range bitplanes {
		bp.DeltaEncode()
	}

	crumbImage := imgtools.ImagePlanarToCrumb(planarImg)
	crumbPlanes := crumbImage.GetPlanes()

	var crumbBins [16]uint64
	var crumbPredictBins [16][16]uint64
	var crumbCount uint64
	var lastCrumb imgtools.Crumb

	for _, plane := range crumbPlanes {
		crumbMtx := plane.GetCrumbs()
		for _, row := range crumbMtx {
			for _, crumb := range row {
				crumbBins[crumb]++
				crumbCount++
			}
		}

		reader, err := comp.NewCrumbReader(&crumbMtx)
		if err != nil {
			panic(err)
		}
		lastCrumb, _ = reader.ReadCrumb()
		for !reader.IsAtEnd() {
			crumb, _ := reader.ReadCrumb()
			crumbPredictBins[lastCrumb][crumb]++
			lastCrumb = crumb
		}
	}

	fmt.Print("\n\nFrequency data:")
	for i, cnt := range crumbBins {
		if i != 15 {
			fmt.Printf("%d,", cnt)
		} else {
			fmt.Printf("%d\n", cnt)
		}
	}
	fmt.Println("\nPrediction data:")
	for j, row := range crumbPredictBins {
		fmt.Printf("%d,", crumbBins[j])
		for i, cnt := range row {
			if i != 15 {
				fmt.Printf("%d,", cnt)
			} else {
				fmt.Printf("%d\n", cnt)
			}
		}
	}
}
