package comp

import (
	"fmt"
	"io"

	"github.com/Kagamiin/pixcrumb/cmd/imgtools"
)

const (
	pc1Name       = "pixcrumb-rle"
	pc1AbbrevName = "pc1"
)

type PixCrumb1Blob struct {
	heightFragments uint8
	widthTiles      uint8
	rleStream       []byte
	dataStream      []byte
}

var _ PixCrumbBlob = &PixCrumb1Blob{}

func (b *PixCrumb1Blob) GetTotalSize() uint64 {
	return uint64(len(b.rleStream) + len(b.dataStream) + 4)
}

type pixCrumb1State struct {
	blob                 PixCrumb1Blob
	crumbReader          CrumbReader
	rleEnc               BitstreamBE
	dataEnc              BitstreamBE
	rleMode              bool
	rleCount             uint16
	modeSwitches         uint64
	literalCrumbsWritten uint64
	rleCrumbsProcessed   uint64
}

var _ PixCrumbCodec = &pixCrumb1State{}

func NewPixCrumb1() PixCrumbEncoder {
	return &pixCrumb1State{}
}

func NewPixCrumb1Decoder(compressedData PixCrumb1Blob) PixCrumbDecoder {
	return &pixCrumb1State{
		blob: compressedData,
	}
}

func (s *pixCrumb1State) GetName() string {
	return pc1Name
}

func (s *pixCrumb1State) GetAbbrevName() string {
	return pc1AbbrevName
}

func (s *pixCrumb1State) Compress(crp *imgtools.CrumbPlane) (blob PixCrumbBlob, err error) {
	wb := crp.GetWidthBpBytes()
	h := crp.GetHeightCrumbs()
	if wb > 255 || h > 255 {
		return nil, fmt.Errorf("%w: rounded pixel dimensions %dx%d exceed max dimensions of 2040x510", ErrImageTooLarge, wb*8, h*2)
	}
	s.blob = PixCrumb1Blob{
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
					//s.crumbReader.Seek(-1, io.SeekCurrent)
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
			s.rleEnc.WriteExpOrderKGolombNumber16(s.rleCount-1, 0)
			s.rleCount = 0
			s.rleMode = false
			s.modeSwitches++
		}
	}

	//fmt.Printf("encoding completed with %d modeswitches, %d literal crumbs written, %d rle crumbs processed, total %d crumbs\n", s.modeSwitches, s.literalCrumbsWritten, s.rleCrumbsProcessed, s.literalCrumbsWritten+s.rleCrumbsProcessed)

	return &s.blob, nil
}

func (s *pixCrumb1State) Decompress() (*imgtools.CrumbPlane, error) {
	panic("not implemented")
}
