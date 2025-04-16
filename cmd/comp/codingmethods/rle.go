package codingmethods

import (
	"errors"
	"io"

	"github.com/Kagamiin/pixcrumb/cmd/imgtools"
)

type expGolombCodedZeroRLECoder struct {
	crumbReader CrumbReader
	codeWriter  BitstreamMSBWriter

	codeReader  BitstreamMSBReader
	crumbWriter CrumbWriter

	golombOrder uint16
}

func NewExpGolombCodedZeroRLECoder(
	encSrc CrumbReader,
	encDest BitstreamMSBWriter,

	decSrc BitstreamMSBReader,
	decDest CrumbWriter,

	golombOrder uint16,
) (CodingMethod, error) {
	if (encSrc == nil) != (encDest == nil) {
		return nil, errors.New("encode source supplied without a destination (or vice-versa)")
	}
	if (decSrc == nil) != (decDest == nil) {
		return nil, errors.New("decode source supplied without a destination (or vice-versa)")
	}
	return &expGolombCodedZeroRLECoder{
		crumbReader: encSrc,
		codeWriter:  encDest,
		codeReader:  decSrc,
		crumbWriter: decDest,
		golombOrder: golombOrder,
	}, nil
}

func (zrc *expGolombCodedZeroRLECoder) DecodeSome() (nCrumbs uint64, bitsRead uint64, err error) {
	if zrc.codeReader == nil || zrc.crumbWriter == nil {
		panic("tried to encode without having supplied encoding source/destination")
	}
	n, err := zrc.codeReader.ReadOrderKExpGolombNumber16(zrc.golombOrder)
	nCrumbs = uint64(n)
	bitsRead = GetNumBitsOrderKExpGolombNumber16(n, zrc.golombOrder)
	zrc.crumbWriter.WriteCrumbs(make([]imgtools.Crumb, n))
	return
}

func (zrc *expGolombCodedZeroRLECoder) EncodeSome() (nCrumbs uint64, bitsWritten uint64, err error) {
	if zrc.codeWriter == nil || zrc.crumbReader == nil {
		panic("tried to decode without having supplied decoding source/destination")
	}
	for !zrc.crumbReader.IsAtEnd() && nCrumbs < 0xFFFF {
		c, err := zrc.crumbReader.ReadCrumb()
		if err != nil {
			return 0, 0, err
		}
		if c != 0 {
			zrc.crumbReader.Seek(-1, io.SeekCurrent)
			break
		}
		nCrumbs++
	}
	zrc.codeWriter.WriteOrderKExpGolombNumber16(uint16(nCrumbs), zrc.golombOrder)
	bitsWritten = GetNumBitsOrderKExpGolombNumber16(uint16(nCrumbs), zrc.golombOrder)
	return
}
