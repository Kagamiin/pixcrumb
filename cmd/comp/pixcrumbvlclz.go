package comp

import (
	"errors"
	"fmt"
	"io"

	"github.com/Kagamiin/pixcrumb/cmd/imgtools"
)

const (
	pcVLCLZName       = "pixcrumb-vlc-lz"
	pcVLCLZAbbrevName = "pclz2"
	pcVLCLZWindowSize = 64
)

type PixCrumbVLCLZBlob struct {
	heightFragments uint8
	widthTiles      uint8
	dataStream      []byte
}

func (b *PixCrumbVLCLZBlob) GetTotalSize() uint64 {
	return uint64(len(b.dataStream) + 2)
}

var _ PixCrumbBlob = &PixCrumbVLCLZBlob{}

type pixCrumbVLCLZState struct {
	blob        PixCrumbVLCLZBlob
	crumbReader CrumbReader
	dataEnc     BitstreamBE
}

func NewPixCrumbVLCLZ() PixCrumbEncoder {
	return &pixCrumbVLCLZState{}
}

func NewPixCrumbVLCLZDecoder(compressedData PixCrumbVLCLZBlob) PixCrumbDecoder {
	return &pixCrumbVLCLZState{
		blob: compressedData,
	}
}

func (s *pixCrumbVLCLZState) Compress(crp *imgtools.CrumbPlane) (blob PixCrumbBlob, err error) {
	wb := crp.GetWidthBpBytes()
	h := crp.GetHeightCrumbs()
	if wb > 255 || h > 255 {
		return nil, fmt.Errorf("%w: rounded pixel dimensions %dx%d exceed max dimensions of 2040x510", ErrImageTooLarge, wb*8, h*2)
	}
	s.blob = PixCrumbVLCLZBlob{
		heightFragments: uint8(h),
		widthTiles:      uint8(wb),
		dataStream:      make([]byte, 0),
	}
	s.dataEnc.data = &s.blob.dataStream
	s.dataEnc.Reset()

	rawData := crp.GetCrumbs()
	s.crumbReader, err = NewCrumbReader(&rawData)
	if err != nil {
		return nil, err
	}

	var cList []imgtools.Crumb = nil
	for !s.crumbReader.IsAtEnd() {
		cList = nil
		for !s.crumbReader.IsAtEnd() {
			length, offset := s.findLZMatch(pcVLCLZWindowSize)
			if length > 0 {
				var literalSize, lzSize uint64
				lzList := s.getLZData(length, offset)
				literalSize = s.dataEnc.GetNumBitsDictCodedCrumbs(lzList, DictLZ)
				literalSize += uint64(DictLZ[TOKEN_END_OF_LITERALS].length)
				lzSize = uint64(DictLZ[TOKEN_END_OF_LITERALS].length)
				lzSize += s.dataEnc.GetNumBitsOrderKExpGolombNumber16(uint16(length-1), 0)
				lzSize += s.dataEnc.GetNumBitsOrderKExpGolombNumber16(uint16(offset-1), 0)
				if lzSize < literalSize {
					cList = append(cList, TOKEN_END_OF_LITERALS)
					s.dataEnc.WriteDictCodedCrumbs(cList, DictLZ)
					if length > 0xFFFF {
						length = 0xFFFF
					}
					s.dataEnc.WriteOrderKExpGolombNumber16(uint16(length-1), 0)
					s.dataEnc.WriteOrderKExpGolombNumber16(uint16(offset-1), 0)
					s.crumbReader.Seek(int64(length), io.SeekCurrent)
					break
				} else {
					cList = append(cList, lzList...)
					s.crumbReader.Seek(int64(length), io.SeekCurrent)
					continue
				}
			}
			c, err := s.crumbReader.ReadCrumb()
			if err != nil {
				return nil, err
			}
			cList = append(cList, c)
		}
	}
	cList = append(cList, TOKEN_END_OF_LITERALS)
	s.dataEnc.WriteDictCodedCrumbs(cList, DictLZ)

	return &s.blob, nil
}

func (s *pixCrumbVLCLZState) findLZMatch(windowSize uint64) (bestLength, bestOffset uint64) {
	//var bestCopiedValues []imgtools.Crumb
	for offs := int64(1); offs < int64(windowSize); offs++ {
		var len int64 = 0
		for {
			dest, err1 := s.crumbReader.PeekCrumbAt(len, true)
			src, err2 := s.crumbReader.PeekCrumbAt(-offs+int64(len%offs), true)
			if errors.Is(err1, ErrCrumbIndexOutOfBounds) || errors.Is(err2, ErrCrumbIndexOutOfBounds) {
				break
			}
			if dest != src {
				break
			}
			len++
		}
		if len > int64(bestLength) {
			bestLength = uint64(len)
			bestOffset = uint64(offs)
		}
	}
	//fmt.Printf("LZ: length %d offset %d - %v\n", bestLength, bestOffset, bestCopiedValues)
	return
}

func (s *pixCrumbVLCLZState) getLZData(length, offset uint64) (cList []imgtools.Crumb) {
	for range length {
		c, err := s.crumbReader.PeekCrumbAt(int64(-offset)+int64(length%offset), true)
		if errors.Is(err, ErrCrumbIndexOutOfBounds) {
			break
		}
		cList = append(cList, c)
	}
	return
}

func (s *pixCrumbVLCLZState) Decompress() (*imgtools.CrumbPlane, error) {
	panic("unimplemented")
}

func (*pixCrumbVLCLZState) GetAbbrevName() string {
	return pcVLCLZAbbrevName
}

func (*pixCrumbVLCLZState) GetName() string {
	return pcVLCLZName
}

var _ PixCrumbCodec = &pixCrumbVLCLZState{}
