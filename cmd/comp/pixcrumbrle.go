package comp

import (
	"fmt"

	"github.com/Kagamiin/pixcrumb/cmd/comp/codingmethods"
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
	blob    PixCrumbRLEBlob
	rleMode bool
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
	rleEnc := codingmethods.NewBitstreamMSBWriter(&s.blob.rleStream)
	dataEnc := codingmethods.NewBitstreamMSBWriter(&s.blob.dataStream)
	s.rleMode = false

	rawData := crp.GetCrumbs()
	crumbReader, err := codingmethods.NewCrumbReader(&rawData)
	if err != nil {
		return nil, err
	}

	literalEncoder, err := codingmethods.NewZeroTerminated4BitCrumbLiteralCoder(crumbReader, dataEnc, nil, nil)
	if err != nil {
		return nil, err
	}

	rleEncoder, err := codingmethods.NewExpGolombCodedZeroRLECoder(crumbReader, rleEnc, nil, nil, 2)
	if err != nil {
		return nil, err
	}

	for !crumbReader.IsAtEnd() {
		if !s.rleMode {
			_, _, err := literalEncoder.EncodeSome()
			if err != nil {
				return nil, err
			}
			s.rleMode = true
		} else {
			_, _, err := rleEncoder.EncodeSome()
			if err != nil {
				return nil, err
			}
			s.rleMode = false
		}
	}

	//fmt.Printf("encoding completed with %d modeswitches, %d literal crumbs written, %d rle crumbs processed, total %d crumbs\n", s.modeSwitches, s.literalCrumbsWritten, s.rleCrumbsProcessed, s.literalCrumbsWritten+s.rleCrumbsProcessed)

	return &s.blob, nil
}

func (s *pixCrumbRLEState) Decompress() (*imgtools.CrumbPlane, error) {
	panic("aaai eu amo a sockettgirl <3")
}
