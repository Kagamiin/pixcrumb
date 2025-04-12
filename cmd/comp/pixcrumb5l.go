package comp

import (
	"errors"
	"fmt"
	"io"

	"github.com/Kagamiin/pixcrumb/cmd/imgtools"
)

const (
	pc5lName       = "pixcrumb-lz"
	pc5lAbbrevName = "pc5l"
)

type PixCrumb5lBlob struct {
	heightFragments uint8
	widthTiles      uint8
	lzStream        []byte
	dataStream      []byte
}

func (b *PixCrumb5lBlob) GetTotalSize() uint64 {
	return uint64(len(b.lzStream) + len(b.dataStream) + 4)
}

var _ PixCrumbBlob = &PixCrumb5lBlob{}

type pixCrumb5lState struct {
	blob        PixCrumb5lBlob
	crumbReader CrumbReader
	lzEnc       BitstreamBE
	dataEnc     BitstreamBE
	lzMode      bool
}

func NewPixCrumb5l() PixCrumbEncoder {
	return &pixCrumb5lState{}
}

func NewPixCrumb5lDecoder(compressedData PixCrumb5lBlob) PixCrumbDecoder {
	return &pixCrumb5lState{
		blob: compressedData,
	}
}

func (s *pixCrumb5lState) Compress(crp *imgtools.CrumbPlane) (blob PixCrumbBlob, err error) {
	wb := crp.GetWidthBpBytes()
	h := crp.GetHeightCrumbs()
	if wb > 255 || h > 255 {
		return nil, fmt.Errorf("%w: rounded pixel dimensions %dx%d exceed max dimensions of 2040x510", ErrImageTooLarge, wb*8, h*2)
	}
	s.blob = PixCrumb5lBlob{
		heightFragments: uint8(h),
		widthTiles:      uint8(wb),
		lzStream:        make([]byte, 0),
		dataStream:      make([]byte, 0),
	}
	s.lzEnc.data = &s.blob.lzStream
	s.dataEnc.data = &s.blob.dataStream
	s.lzMode = false
	s.lzEnc.Reset()
	s.dataEnc.Reset()

	rawData := crp.GetCrumbs()
	s.crumbReader, err = NewCrumbReader(&rawData)
	if err != nil {
		return nil, err
	}

	for !s.crumbReader.IsAtEnd() {
		if !s.lzMode {
			var cList []imgtools.Crumb
			for !s.crumbReader.IsAtEnd() {
				c, err := s.crumbReader.ReadCrumb()
				if err != nil {
					return nil, err
				}
				cList = append(cList, c)
				if c == 0 {
					s.lzMode = true
					break
				}
			}
			s.dataEnc.WriteCrumbs(cList)
		} else {
			length, offset := s.findLZMatch(16)
			if length > 0xFFFF {
				length = 0xFFFF
			}
			s.lzEnc.WriteExpOrderKGolombNumber16(uint16(length), 0)
			if length > 0 && offset > 0 {
				s.lzEnc.WriteExpOrderKGolombNumber16(uint16(offset-1), 0)
				s.crumbReader.Seek(int64(length), io.SeekCurrent)
			}
			s.lzMode = false
		}
	}

	return &s.blob, nil
}

func (s *pixCrumb5lState) findLZMatch(windowSize uint64) (bestLength, bestOffset uint64) {
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

func (s *pixCrumb5lState) Decompress() (*imgtools.CrumbPlane, error) {
	panic("unimplemented")
}

func (*pixCrumb5lState) GetAbbrevName() string {
	return pc5lAbbrevName
}

func (*pixCrumb5lState) GetName() string {
	return pc5lName
}

var _ PixCrumbCodec = &pixCrumb5lState{}
