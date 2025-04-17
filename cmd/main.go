package main

import (
	// "flag"
	"fmt"
	"image"
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

		compPlaneBlobs, err := compressImageIntoPixCrumbBlobs(img, comp.NewPixCrumbRLEEncoder())
		if err != nil {
			log.Println(err)
		}
		_ = compPlaneBlobs
		// TODO: decompress and save image
	}
}

func compressImageIntoPixCrumbBlobs(img image.PalettedImage, codec comp.PixCrumbEncoder) ([]comp.PixCrumbBlob, error) {
	planarImg, err := imgtools.NewPlanarImage(img)
	if err != nil {
		return nil, err
	}
	bitplanes := planarImg.GetBitplanes()

	for _, bp := range bitplanes {
		bp.DeltaEncode()
	}

	crumbImage := imgtools.ImagePlanarToCrumb(planarImg)
	crumbPlanes := crumbImage.GetPlanes()

	var compressedBlobs []comp.PixCrumbBlob

	var totalSizeRaw, totalSizeComp uint64
	fmt.Printf("\nUsing method %s:\n", codec.GetName())
	for i, crp := range crumbPlanes {
		rawSize := bitplanes[i].GetTotalSize()
		comp, err := codec.Compress(&crp)
		if err != nil {
			return nil, fmt.Errorf("error while encoding BP%d: %w", i, err)
		}
		compSize := comp.GetTotalSize()
		fmt.Printf("BP%d raw size: %d bytes, compressed to %d bytes (ratio: %.03f)\n", i, rawSize, compSize, float64(compSize)/float64(rawSize))
		totalSizeRaw += rawSize
		totalSizeComp += compSize
		compressedBlobs = append(compressedBlobs, comp)
	}
	fmt.Printf("Total: raw size %d bytes, compressed to %d bytes (ratio: %.03f)\n\n", totalSizeRaw, totalSizeComp, float64(totalSizeComp)/float64(totalSizeRaw))
	return compressedBlobs, nil
}

func handle(err error) {
	if err != nil {
		log.Fatal("error: ", err)
	}
}
