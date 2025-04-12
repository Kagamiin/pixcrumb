package comp

import (
	"errors"
	"fmt"
	"io"

	"github.com/Kagamiin/pixcrumb/cmd/imgtools"
)

type pixCrumb3Mode int

const (
	PC3_SINGLE_ZERO pixCrumb3Mode = iota
	PC3_ZERO_RLE
	PC3_LITERAL_RLE
	PC3_LITERAL
)

const (
	pc3Name       = "pixcrumb3"
	pc3AbbrevName = "pc3"
)

type PixCrumb3Blob struct {
	heightFragments uint8
	widthTiles      uint8
	commandStream   []byte
	dataStream      []byte
}

var _ PixCrumbBlob = &PixCrumb3Blob{}

func (b *PixCrumb3Blob) GetTotalSize() uint64 {
	return uint64(len(b.commandStream) + len(b.dataStream) + 4)
}

type pixCrumb3State struct {
	blob            PixCrumb3Blob
	crumbReader     CrumbReader
	commandEnc      BitstreamBE
	dataEnc         BitstreamBE
	mode            pixCrumb3Mode
	rleCount        uint16
	modeUsageStats  [4]uint64
	modeSwitchStats [4][4]uint64
}

var _ PixCrumbCodec = &pixCrumb3State{}

func NewPixCrumb3() PixCrumbEncoder {
	return &pixCrumb3State{}
}

func NewPixCrumb3Decoder(compressedData PixCrumb3Blob) PixCrumbDecoder {
	return &pixCrumb3State{
		blob: compressedData,
	}
}

func (s *pixCrumb3State) GetName() string {
	return pc3Name
}

func (s *pixCrumb3State) GetAbbrevName() string {
	return pc3AbbrevName
}

func (s *pixCrumb3State) Compress(crp *imgtools.CrumbPlane) (blob PixCrumbBlob, err error) {
	wb := crp.GetWidthBpBytes()
	//w := crp.GetWidthCrumbs()
	h := crp.GetHeightCrumbs()
	if wb > 255 || h > 255 {
		return nil, fmt.Errorf("%w: rounded pixel dimensions %dx%d exceed max dimensions of 2040x510", ErrImageTooLarge, wb*8, h*2)
	}
	s.blob = PixCrumb3Blob{
		heightFragments: uint8(h),
		widthTiles:      uint8(wb),
		commandStream:   make([]byte, 0),
		dataStream:      make([]byte, 0),
	}
	s.commandEnc.data = &s.blob.commandStream
	s.dataEnc.data = &s.blob.dataStream
	s.mode = -1
	s.rleCount = 0
	s.commandEnc.Reset()
	s.dataEnc.Reset()

	rawData := crp.GetCrumbs()
	s.crumbReader, err = NewCrumbReader(&rawData)
	if err != nil {
		return nil, err
	}

	for !s.crumbReader.IsAtEnd() {
		s.mode, err = s.determineNextMode()
		if err != nil {
			return nil, err
		}

		err = s.executeCurrentMode()
		if err != nil {
			return nil, err
		}
	}

	// PRINT STATISTICS ABOUT MODE USAGES
	{
		var totalModeSwitches uint64
		var totalModeUsages uint64
		var usagePct [4]float64

		for _, usage := range s.modeUsageStats {
			totalModeUsages += usage
		}
		for mode, usage := range s.modeUsageStats {
			usagePct[mode] = float64(usage) / float64(totalModeUsages) * 100
		}

		fmt.Println("Mode stats!  | ->0  ->1  ->2  ->3   Usage    %")
		fmt.Println("------------------------------------------------")
		for mode, stats := range s.modeSwitchStats {
			fmt.Printf("Mode %d stats:", mode)
			for _, v := range stats {
				fmt.Printf(" %4d", v)
				totalModeSwitches += v
			}
			fmt.Printf("%8d%6.02f%%\n", s.modeUsageStats[mode], usagePct[mode])
		}
	}

	return &s.blob, nil
}

func (s *pixCrumb3State) executeCurrentMode() (errOut error) {
	var rleValue imgtools.Crumb = 0
	switch s.mode {
	case PC3_SINGLE_ZERO:
		_, err := s.crumbReader.Seek(1, io.SeekCurrent)
		if err != nil {
			return err
		}
	case PC3_LITERAL_RLE:
		c, err := s.crumbReader.PeekCrumb()
		if err != nil {
			return err
		}
		rleValue = c
		s.dataEnc.WriteBits(uint64(c), 4)
		fallthrough
	case PC3_ZERO_RLE:
		s.rleCount = 0
		for range 0xFFFF {
			if s.crumbReader.IsAtEnd() {
				break
			}
			c, err := s.crumbReader.ReadCrumb()
			if err != nil {
				return err
			}
			if c != rleValue {
				s.crumbReader.Seek(-1, io.SeekCurrent)
				break
			}
			s.rleCount++
		}
		if s.rleCount < 3 {
			panic("RLE mode invoked with less than 3 consecutive equal values")
		}
		s.commandEnc.WriteExpOrderKGolombNumber16(s.rleCount-3, 1)
	case PC3_LITERAL:
		var cList []imgtools.Crumb
		for !s.crumbReader.IsAtEnd() {
			c, err := s.crumbReader.ReadCrumb()
			if err != nil {
				return err
			}
			cList = append(cList, c)
			if c == 0 {
				break
			}
		}
		s.dataEnc.WriteCrumbs(cList)
	}
	return nil
}

func (s *pixCrumb3State) determineNextMode() (mode pixCrumb3Mode, err error) {
	oldMode := s.mode
	cList, err := s.crumbReader.PeekNCrumbs(3)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return -1, err
	}
	if len(cList) == 0 {
		panic(io.ErrUnexpectedEOF)
	}
	if len(cList) == 3 {
		if cList[0] == 0 && (cList[1] != 0 || cList[2] != 0) {
			// We have one or two zeroes in sequence; encode a single zero
			mode = PC3_SINGLE_ZERO
		} else if cList[0] == 0 && cList[1] == 0 && cList[2] == 0 {
			// We have at least 3 zeroes in sequence; encode a Zero-RLE sequence
			mode = PC3_ZERO_RLE
		} else if cList[0] != 0 && cList[1] == cList[0] && cList[2] == cList[1] {
			// We have three of the same literal; encode literal RLE
			mode = PC3_LITERAL_RLE
		} else {
			// We have three literals; encode a literal sequence
			mode = PC3_LITERAL
		}
	} else {
		// There are only one or two crumbs left to encode; the single value modes are more efficient here.
		if cList[0] == 0 {
			mode = PC3_SINGLE_ZERO
		} else {
			mode = PC3_LITERAL
		}
	}
	s.commandEnc.WriteBits(uint64(mode), 2)
	if oldMode != -1 {
		s.modeSwitchStats[oldMode][mode]++
	}
	s.modeUsageStats[mode]++
	return mode, nil
}

func (s *pixCrumb3State) Decompress() (*imgtools.CrumbPlane, error) {
	panic("not implemented")
}
