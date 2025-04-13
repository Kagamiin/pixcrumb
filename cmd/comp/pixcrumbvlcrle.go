package comp

import (
	"fmt"
	"io"

	"github.com/Kagamiin/pixcrumb/cmd/imgtools"
)

const (
	pcVLCRLEName       = "pixcrumb-vlc-rle"
	pcVLCRLEAbbrevName = "pcrle2"
)

type PixCrumbVLCRLEBlob struct {
	heightFragments uint8
	widthTiles      uint8
	dataStream      []byte
}

var _ PixCrumbBlob = &PixCrumbVLCRLEBlob{}

func (b *PixCrumbVLCRLEBlob) GetTotalSize() uint64 {
	return uint64(len(b.dataStream) + 2)
}

type pixCrumbVLCRLEState struct {
	blob        PixCrumbVLCRLEBlob
	crumbReader CrumbReader
	dataEnc     BitstreamBE
	rleMode     bool
	rleCount    uint16

	modeSwitches         uint64
	literalCrumbsWritten uint64
	rleCrumbsProcessed   uint64
}

var _ PixCrumbCodec = &pixCrumbVLCRLEState{}

func NewPixCrumbVLCRLE() PixCrumbEncoder {
	return &pixCrumbVLCRLEState{}
}

func NewPixCrumbVLCRLEDecoder(compressedData PixCrumbVLCRLEBlob) PixCrumbDecoder {
	return &pixCrumbVLCRLEState{
		blob: compressedData,
	}
}

func (s *pixCrumbVLCRLEState) GetName() string {
	return pcVLCRLEName
}

func (s *pixCrumbVLCRLEState) GetAbbrevName() string {
	return pcVLCRLEAbbrevName
}

func (s *pixCrumbVLCRLEState) Compress(crp *imgtools.CrumbPlane) (blob PixCrumbBlob, err error) {
	wb := crp.GetWidthBpBytes()
	h := crp.GetHeightCrumbs()
	if wb > 255 || h > 255 {
		return nil, fmt.Errorf("%w: rounded pixel dimensions %dx%d exceed max dimensions of 2040x510", ErrImageTooLarge, wb*8, h*2)
	}
	s.blob = PixCrumbVLCRLEBlob{
		heightFragments: uint8(h),
		widthTiles:      uint8(wb),
		dataStream:      make([]byte, 0),
	}
	s.dataEnc.data = &s.blob.dataStream
	s.rleMode = false
	s.rleCount = 0
	s.dataEnc.Reset()
	s.modeSwitches = 0
	s.literalCrumbsWritten = 0
	s.rleCrumbsProcessed = 0

	rawData := crp.GetCrumbs()
	s.crumbReader, err = NewCrumbReader(&rawData)
	if err != nil {
		return nil, err
	}

	for !s.crumbReader.IsAtEnd() {
		if !s.rleMode {
			for !s.crumbReader.IsAtEnd() {
				c, err := s.crumbReader.ReadCrumb()
				if err != nil {
					return nil, err
				}
				s.dataEnc.WriteDictEntry(DictRLE[c])
				if c == 0 {
					break
				}
				s.literalCrumbsWritten++
			}
			s.rleCount = 1
			s.rleMode = true
			s.modeSwitches++
		} else {
			for !s.crumbReader.IsAtEnd() && s.rleCount < 0xFFFF {
				c, err := s.crumbReader.ReadCrumb()
				if err != nil {
					return nil, err
				}
				if c != 0 {
					s.crumbReader.Seek(-1, io.SeekCurrent)
					break
				}
				s.rleCount++
				s.rleCrumbsProcessed++
			}
			s.dataEnc.WriteOrderKExpGolombNumber16(s.rleCount-1, 0)
			s.rleCount = 0
			s.rleMode = false
			s.modeSwitches++
		}
	}

	//fmt.Printf("encoding completed with %d modeswitches, %d literal crumbs written, %d rle crumbs processed, total %d crumbs\n", s.modeSwitches, s.literalCrumbsWritten, s.rleCrumbsProcessed, s.literalCrumbsWritten+s.rleCrumbsProcessed)

	return &s.blob, nil
}

func (s *pixCrumbVLCRLEState) Decompress() (*imgtools.CrumbPlane, error) {
	panic("not implemented")
}
