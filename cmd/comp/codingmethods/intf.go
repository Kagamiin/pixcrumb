package codingmethods

import "github.com/Kagamiin/pixcrumb/cmd/imgtools"

type CodingMethod interface {
	EncodeSome() (nCrumbs uint64, bitsWritten uint64, err error)
	DecodeSome() (nCrumbs uint64, bitsRead uint64, err error)
}

type bitstreamMSBPeeker interface {
	Reset()
	PeekBit() uint8
	Tell() int64
	BitsLeft() int64
	Seek(offset int64, whence int) (int64, error)
	GetData() *[]byte
}

type BitstreamMSBReader interface {
	bitstreamMSBPeeker
	ReadBit() (res uint8, err error)
	ReadBits(count uint) (res uint64, err error)
	ReadOrderKExpGolombNumber16(order uint16) (uint16, error)
}

type BitstreamMSBWriter interface {
	bitstreamMSBPeeker
	PokeBit(bit uint8)
	WriteBit(bit uint8)
	WriteBits(val uint64, count uint)
	WriteCrumbs(cList []imgtools.Crumb)
	WriteDictEntry(word bitDictWord)
	WriteDictCodedCrumbs(cList []imgtools.Crumb, dict BitDict)
	WriteOrderKExpGolombNumber16(value uint16, order uint16)
}
