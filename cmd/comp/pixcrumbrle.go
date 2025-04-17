package comp

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/Kagamiin/pixcrumb/cmd/comp/codingmethods"
	"github.com/Kagamiin/pixcrumb/cmd/imgtools"
)

const (
	pcRLEName       = "pixcrumb-rle"
	pcRLEAbbrevName = "pcrle"
)

type pixCrumbRLEBlob struct {
	heightCrumbs uint8
	widthTiles   uint8
	rleStream    []byte
	dataStream   []byte
}

var _ PixCrumbBlob = &pixCrumbRLEBlob{}

func (b *pixCrumbRLEBlob) GetTotalSize() uint64 {
	return uint64(len(b.rleStream) + len(b.dataStream) + 4)
}

func (b *pixCrumbRLEBlob) GetHeightCrumbs() uint8 {
	return b.heightCrumbs
}

func (b *pixCrumbRLEBlob) GetWidthTiles() uint8 {
	return b.widthTiles
}

func (b *pixCrumbRLEBlob) Marshal() ([]byte, error) {
	result := make([]byte, b.GetTotalSize())
	writer := bytes.NewBuffer(result)
	writer.WriteByte(b.heightCrumbs)
	writer.WriteByte(b.widthTiles)
	writer.Write(binary.LittleEndian.AppendUint16(nil, uint16(len(b.rleStream)+2)))
	writer.Write(b.rleStream)
	writer.Write(b.dataStream)
	if writer.Len() != int(b.GetTotalSize()) {
		panic("number of written bytes does not match buffer size!")
	}
	return result, nil
}

func (b *pixCrumbRLEBlob) Unmarshal(data []byte) error {
	reader := bytes.NewBuffer(data)
	if len(data) < 4 {
		return ErrBlobDataInvalid
	}

	b.heightCrumbs, _ = reader.ReadByte()
	b.widthTiles, _ = reader.ReadByte()
	u16le := make([]byte, 2)
	_, _ = reader.Read(u16le)
	dataBlobOffset := binary.LittleEndian.Uint16(u16le)
	rleBlobLen := dataBlobOffset - 4
	dataBlobLen := len(data) - int(dataBlobOffset)

	b.rleStream = make([]byte, rleBlobLen)
	b.dataStream = make([]byte, dataBlobLen)
	n, err := reader.Read(b.rleStream)
	if err != nil || n != int(rleBlobLen) {
		return fmt.Errorf("%w: end of data reached while reading RLE blob", ErrBlobDataInconsistent)
	}
	n, err = reader.Read(b.dataStream)
	if err != nil || n != int(dataBlobLen) {
		return fmt.Errorf("%w: end of data reached while reading data blob", ErrBlobDataInconsistent)
	}
	return nil
}

type pixCrumbRLEState struct {
	blob    pixCrumbRLEBlob
	rleMode bool
}

var _ PixCrumbCodec = &pixCrumbRLEState{}

func NewPixCrumbRLEEncoder() PixCrumbEncoder {
	return &pixCrumbRLEState{}
}

func NewPixCrumbRLEDecoder(pcBlob PixCrumbBlob) (PixCrumbDecoder, error) {
	var result pixCrumbRLEState
	if err := result.LoadBlob(pcBlob); err != nil {
		return nil, err
	}
	return &result, nil
}

func NewPixCrumbRLE() PixCrumbCodec {
	return &pixCrumbRLEState{}
}

func (s *pixCrumbRLEState) GetName() string {
	return pcRLEName
}

func (s *pixCrumbRLEState) GetAbbrevName() string {
	return pcRLEAbbrevName
}

func (s *pixCrumbRLEState) LoadBlob(pcBlob PixCrumbBlob) error {
	if b, ok := pcBlob.(*pixCrumbRLEBlob); !ok {
		return fmt.Errorf("cannot load blob into PixCrumbRLE: %w", ErrWrongBlogTypeForCodec)
	} else {
		s.blob = *b
	}
	return nil
}

func (s *pixCrumbRLEState) Compress(crp *imgtools.CrumbPlane) (blob PixCrumbBlob, err error) {
	wb := crp.GetWidthBpBytes()
	h := crp.GetHeightCrumbs()
	if wb > 255 || h > 255 {
		return nil, fmt.Errorf("%w: rounded pixel dimensions %dx%d exceed max dimensions of 2040x510", ErrImageTooLarge, wb*8, h*2)
	}
	s.blob = pixCrumbRLEBlob{
		heightCrumbs: uint8(h),
		widthTiles:   uint8(wb),
		rleStream:    make([]byte, 0),
		dataStream:   make([]byte, 0),
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
	rleDec := codingmethods.NewBitstreamMSBReader(&s.blob.rleStream)
	dataDec := codingmethods.NewBitstreamMSBReader(&s.blob.dataStream)
	s.rleMode = false

	crumbWriter := codingmethods.NewCrumbWriter(uint64(s.blob.widthTiles) * 4)

	literalDecoder, err := codingmethods.NewZeroTerminated4BitCrumbLiteralCoder(nil, nil, dataDec, crumbWriter)
	if err != nil {
		return nil, err
	}

	rleDecoder, err := codingmethods.NewExpGolombCodedZeroRLECoder(nil, nil, rleDec, crumbWriter, 2)
	if err != nil {
		return nil, err
	}

	for crumbWriter.GetHeightCrumbs() < int(s.blob.heightCrumbs) && !crumbWriter.IsLengthAligned() {
		if !s.rleMode {
			_, _, err := literalDecoder.DecodeSome()
			if err != nil {
				return nil, err
			}
			s.rleMode = true
		} else {
			_, _, err := rleDecoder.DecodeSome()
			if err != nil {
				return nil, err
			}
			s.rleMode = false
		}
	}

	crumbMtx, err := crumbWriter.GetCrumbMatrix()
	if err != nil {
		return nil, err
	}

	return imgtools.MakeCrumbPlane(crumbMtx), nil
}
