package comp

import (
	"errors"

	"github.com/Kagamiin/pixcrumb/cmd/imgtools"
)

var (
	ErrImageTooLarge         = errors.New("image too big")
	ErrBlobDataInvalid       = errors.New("blob data is invalid")
	ErrBlobDataInconsistent  = errors.New("blob data has inconsistencies")
	ErrWrongBlogTypeForCodec = errors.New("wrong blob type for this codec")
)

type PixCrumbBlob interface {
	GetTotalSize() uint64
	GetHeightCrumbs() uint8
	GetWidthTiles() uint8
	Marshal() ([]byte, error)
	Unmarshal(data []byte) error
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
	LoadBlob(PixCrumbBlob) error
	Decompress() (*imgtools.CrumbPlane, error)
}

type PixCrumbCodec interface {
	PixCrumbEncoder
	PixCrumbDecoder
}
