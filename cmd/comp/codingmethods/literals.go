package codingmethods

import (
	"errors"

	"github.com/Kagamiin/pixcrumb/cmd/imgtools"
)

type zeroTerminated4BitCrumbLiteralCoder struct {
	crumbReader   CrumbReader
	literalWriter BitstreamMSBWriter

	literalReader BitstreamMSBReader
	crumbWriter   CrumbWriter
}

func NewZeroTerminated4BitCrumbLiteralCoder(
	encSrc CrumbReader,
	encDest BitstreamMSBWriter,

	decSrc BitstreamMSBReader,
	decDest CrumbWriter,
) (CodingMethod, error) {
	if (encSrc == nil) != (encDest == nil) {
		return nil, errors.New("encode source supplied without a destination (or vice-versa)")
	}
	if (decSrc == nil) != (decDest == nil) {
		return nil, errors.New("decode source supplied without a destination (or vice-versa)")
	}
	return &zeroTerminated4BitCrumbLiteralCoder{
		crumbReader:   encSrc,
		literalWriter: encDest,
		literalReader: decSrc,
		crumbWriter:   decDest,
	}, nil
}

var _ CodingMethod = &zeroTerminated4BitCrumbLiteralCoder{}

func (clc *zeroTerminated4BitCrumbLiteralCoder) EncodeSome() (nCrumbs uint64, bitsWritten uint64, err error) {
	if clc.crumbReader == nil || clc.literalWriter == nil {
		panic("tried to encode without having supplied encoding source/destination")
	}
	var cList []imgtools.Crumb
	for !clc.crumbReader.IsAtEnd() {
		c, err := clc.crumbReader.ReadCrumb()
		if err != nil {
			return 0, 0, err
		}
		cList = append(cList, c)
		if c == 0 {
			break
		}
	}
	clc.literalWriter.WriteCrumbs(cList)
	return uint64(len(cList)) - 1, uint64(len(cList)) * 4, nil
}

func (clc *zeroTerminated4BitCrumbLiteralCoder) DecodeSome() (nCrumbs uint64, bitsRead uint64, err error) {
	if clc.literalReader == nil || clc.crumbWriter == nil {
		panic("tried to encode without having supplied encoding source/destination")
	}
	var cList []imgtools.Crumb
	for clc.literalReader.BitsLeft() > 4 {
		c, err := clc.literalReader.ReadBits(4)
		if err != nil {
			return 0, 0, err
		}
		if c == 0 {
			break
		}
		cList = append(cList, imgtools.Crumb(c))
	}
	clc.crumbWriter.WriteCrumbs(cList)
	return uint64(len(cList)), uint64(len(cList)+1) * 4, nil
}
