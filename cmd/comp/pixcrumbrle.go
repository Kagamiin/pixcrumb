package comp

import (
	"fmt"
	"io"

	"github.com/Kagamiin/pixcrumb/cmd/imgtools"
)

const (
	pcRLEName       = "pixcrumb-rle"
	pcRLEAbbrevName = "pcrle"
)

type PixCrumbRLEBlob struct {
	heightFragments uint8
	widthTiles      uint8
	rleStream       []byte
	dataStream      []byte
}

var _ PixCrumbBlob = &PixCrumbRLEBlob{}

func (b *PixCrumbRLEBlob) GetTotalSize() uint64 {
	return uint64(len(b.rleStream) + len(b.dataStream) + 4)
}

type pixCrumbRLEState struct {
	blob        PixCrumbRLEBlob
	crumbReader CrumbReader
	rleEnc      BitstreamBE
	dataEnc     BitstreamBE
	rleMode     bool
	rleCount    uint16

	modeSwitches         uint64
	literalCrumbsWritten uint64
	rleCrumbsProcessed   uint64
}

var _ PixCrumbCodec = &pixCrumbRLEState{}

func NewPixCrumbRLE() PixCrumbEncoder {
	return &pixCrumbRLEState{}
}

func NewPixCrumbRLEDecoder(compressedData PixCrumbRLEBlob) PixCrumbDecoder {
	return &pixCrumbRLEState{
		blob: compressedData,
	}
}

func (s *pixCrumbRLEState) GetName() string {
	return pcRLEName
}

func (s *pixCrumbRLEState) GetAbbrevName() string {
	return pcRLEAbbrevName
}

func (s *pixCrumbRLEState) Compress(crp *imgtools.CrumbPlane) (blob PixCrumbBlob, err error) {
	wb := crp.GetWidthBpBytes()
	h := crp.GetHeightCrumbs()
	if wb > 255 || h > 255 {
		return nil, fmt.Errorf("%w: rounded pixel dimensions %dx%d exceed max dimensions of 2040x510", ErrImageTooLarge, wb*8, h*2)
	}
	s.blob = PixCrumbRLEBlob{
		heightFragments: uint8(h),
		widthTiles:      uint8(wb),
		rleStream:       make([]byte, 0),
		dataStream:      make([]byte, 0),
	}
	s.rleEnc.data = &s.blob.rleStream
	s.dataEnc.data = &s.blob.dataStream
	s.rleMode = false
	s.rleCount = 0
	s.rleEnc.Reset()
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
			var cList []imgtools.Crumb = nil
			for !s.crumbReader.IsAtEnd() {
				c, err := s.crumbReader.ReadCrumb()
				if err != nil {
					return nil, err
				}
				cList = append(cList, c)
				if c == 0 {
					break
				}
				s.literalCrumbsWritten++
			}
			s.rleCount = 1
			s.rleMode = true
			s.modeSwitches++
			s.dataEnc.WriteCrumbs(cList)
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
			s.rleEnc.WriteOrderKExpGolombNumber16(s.rleCount-1, 0)
			s.rleCount = 0
			s.rleMode = false
			s.modeSwitches++
		}
	}

	//fmt.Printf("encoding completed with %d modeswitches, %d literal crumbs written, %d rle crumbs processed, total %d crumbs\n", s.modeSwitches, s.literalCrumbsWritten, s.rleCrumbsProcessed, s.literalCrumbsWritten+s.rleCrumbsProcessed)

	return &s.blob, nil
}

func (s *pixCrumbRLEState) Decompress() (*imgtools.CrumbPlane, error) {
	panic("not implemented")
}
