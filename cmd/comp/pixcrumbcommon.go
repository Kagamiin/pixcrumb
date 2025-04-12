package comp

import (
	"errors"

	"github.com/Kagamiin/pixcrumb/cmd/imgtools"
)

var ErrImageTooLarge = errors.New("image too big")

type PixCrumbBlob interface {
	GetTotalSize() uint64
}

type PixCrumbCodecBase interface {
	GetName() string
	GetAbbrevName() string
}

type PixCrumbEncoder interface {
	PixCrumbCodecBase
	Compress(crp *imgtools.CrumbPlane) (PixCrumbBlob, error)
}

type PixCrumbDecoder interface {
	PixCrumbCodecBase
	Decompress() (*imgtools.CrumbPlane, error)
}

type PixCrumbCodec interface {
	PixCrumbEncoder
	PixCrumbDecoder
}
