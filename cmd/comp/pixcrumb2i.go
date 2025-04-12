package comp

import (
	"errors"
	"fmt"
	"io"

	"github.com/Kagamiin/pixcrumb/cmd/imgtools"
)

type pixCrumb2iMode int

const (
	PC2I_SINGLE_ZERO pixCrumb2iMode = iota
	PC2I_ZERO_RLE
	PC2I_LITERAL
	PC2I_INVALID pixCrumb2iMode = -1
)

const (
	pc2iName       = "pixCrumb2i"
	pc2iAbbrevName = "pc2i"
)

type PixCrumb2iBlob struct {
	heightFragments uint8
	widthTiles      uint8
	commandStream   []byte
	dataStream      []byte
}

var _ PixCrumbBlob = &PixCrumb2iBlob{}

func (b *PixCrumb2iBlob) GetTotalSize() uint64 {
	return uint64(len(b.commandStream) + len(b.dataStream) + 4)
}

type pixCrumb2iState struct {
	blob            PixCrumb2iBlob
	crumbReader     CrumbReader
	commandEnc      BitstreamBE
	dataEnc         BitstreamBE
	mode            pixCrumb2iMode
	rleCount        uint16
	modeUsageStats  [3]uint64
	modeSwitchStats [3][3]uint64
}

var _ PixCrumbCodec = &pixCrumb2iState{}

var pixCrumb2iModeSwitchMap = map[pixCrumb2iMode][2]pixCrumb2iMode{
	PC2I_SINGLE_ZERO: {PC2I_SINGLE_ZERO, PC2I_LITERAL},
	PC2I_ZERO_RLE:    {PC2I_INVALID, PC2I_INVALID},
	PC2I_LITERAL:     {PC2I_ZERO_RLE, PC2I_LITERAL},
}

var pixCrumb2iModeSignalMap = map[[2]pixCrumb2iMode]int{
	{PC2I_SINGLE_ZERO, PC2I_SINGLE_ZERO}: 0,
	{PC2I_SINGLE_ZERO, PC2I_LITERAL}:     1,
	{PC2I_LITERAL, PC2I_ZERO_RLE}:        0,
	{PC2I_LITERAL, PC2I_LITERAL}:         1,
}

var pixCrumb2iReverseModeSwitchMap = map[int][3]pixCrumb2iMode{
	0: {PC2I_SINGLE_ZERO, PC2I_INVALID, PC2I_ZERO_RLE},
	1: {PC2I_LITERAL, PC2I_INVALID, PC2I_LITERAL},
}

func NewPixCrumb2i() PixCrumbEncoder {
	return &pixCrumb2iState{}
}

func NewPixCrumb2iDecoder(compressedData PixCrumb2iBlob) PixCrumbDecoder {
	return &pixCrumb2iState{
		blob: compressedData,
	}
}

func (s *pixCrumb2iState) GetName() string {
	return pc2iName
}

func (s *pixCrumb2iState) GetAbbrevName() string {
	return pc2iAbbrevName
}

func (s *pixCrumb2iState) Compress(crp *imgtools.CrumbPlane) (blob PixCrumbBlob, err error) {
	wb := crp.GetWidthBpBytes()
	//w := crp.GetWidthCrumbs()
	h := crp.GetHeightCrumbs()
	if wb > 255 || h > 255 {
		return nil, fmt.Errorf("%w: rounded pixel dimensions %dx%d exceed max dimensions of 2040x510", ErrImageTooLarge, wb*8, h*2)
	}
	s.blob = PixCrumb2iBlob{
		heightFragments: uint8(h),
		widthTiles:      uint8(wb),
		commandStream:   make([]byte, 0),
		dataStream:      make([]byte, 0),
	}
	s.commandEnc.data = &s.blob.commandStream
	s.dataEnc.data = &s.blob.dataStream
	s.mode = PC2I_LITERAL
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
		var usagePct [3]float64

		for _, usage := range s.modeUsageStats {
			totalModeUsages += usage
		}
		for mode, usage := range s.modeUsageStats {
			usagePct[mode] = float64(usage) / float64(totalModeUsages) * 100
		}

		fmt.Println("Mode stats!  | ->0  ->1  ->2   Usage    %")
		fmt.Println("--------------------------------------------")
		for mode, stats := range s.modeSwitchStats {
			fmt.Printf("Mode %d stats:", mode)
			for _, v := range stats {
				fmt.Printf(" %4d", v)
				totalModeSwitches += v
			}
			fmt.Printf("%8d %6.02f%%\n", s.modeUsageStats[mode], usagePct[mode])
		}
	}

	return &s.blob, nil
}

func (s *pixCrumb2iState) executeCurrentMode() (errOut error) {
	switch s.mode {
	case PC2I_SINGLE_ZERO:
		_, err := s.crumbReader.Seek(1, io.SeekCurrent)
		if err != nil {
			return err
		}
	case PC2I_ZERO_RLE:
		s.rleCount = 0
		for range 0xFFFF {
			if s.crumbReader.IsAtEnd() {
				break
			}
			c, err := s.crumbReader.ReadCrumb()
			if err != nil {
				return err
			}
			if c != 0 {
				s.crumbReader.Seek(-1, io.SeekCurrent)
				break
			}
			s.rleCount++
		}
		if s.rleCount < 2 {
			panic("zero-RLE mode invoked with less than 2 consecutive zero values")
		}
		s.commandEnc.WriteExpOrderKGolombNumber16(s.rleCount-2, 2)
	case PC2I_LITERAL:
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

func (s *pixCrumb2iState) determineNextMode() (mode pixCrumb2iMode, err error) {
	oldMode := s.mode
	mode = PC2I_LITERAL
	if oldMode == PC2I_ZERO_RLE {
		s.modeSwitchStats[oldMode][mode]++
		s.modeUsageStats[mode]++
		return
	}
	cList, err := s.crumbReader.PeekNCrumbs(2)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return -1, err
	}
	if len(cList) == 0 {
		panic(io.ErrUnexpectedEOF)
	}
	if len(cList) == 2 {
		possibleModes := pixCrumb2iModeSwitchMap[oldMode]
		switch possibleModes[0] {
		case PC2I_SINGLE_ZERO:
			if cList[0] == 0 {
				mode = PC2I_SINGLE_ZERO
			}
		case PC2I_ZERO_RLE:
			if cList[0] == 0 && cList[1] == 0 {
				mode = PC2I_ZERO_RLE
			}
		default:
			panic("unreachable code")
		}
	}
	bitToWrite, ok := pixCrumb2iModeSignalMap[[2]pixCrumb2iMode{oldMode, mode}]
	if ok {
		s.commandEnc.WriteBits(uint64(bitToWrite), 1)
	} else {
		panic(fmt.Sprintf("invalid mode transition: %d -> %d", oldMode, mode))
	}
	s.modeSwitchStats[oldMode][mode]++
	s.modeUsageStats[mode]++
	return mode, nil
}

func (s *pixCrumb2iState) Decompress() (*imgtools.CrumbPlane, error) {
	panic("not implemented")
}
