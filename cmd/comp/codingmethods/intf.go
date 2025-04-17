package codingmethods

import "github.com/Kagamiin/pixcrumb/cmd/imgtools"

type CodingMethod interface {
	EncodeSome() (nCrumbs uint64, bitsWritten uint64, err error)
	DecodeSome() (nCrumbs uint64, bitsRead uint64, err error)
}

type CrumbPeeker interface {
	Length() uint64
	Seek(offset int64, whence int) (int64, error)
	Tell() int64
	PeekCrumb() (imgtools.Crumb, error)
	PeekNCrumbs(n uint64) ([]imgtools.Crumb, error)
	PeekCrumbAt(offset int64, relative bool) (imgtools.Crumb, error)
	PeekNCrumbsAt(n uint64, offset int64, relative bool) ([]imgtools.Crumb, error)
	IsLengthAligned() bool
	IsAtEnd() bool
	GetHeightCrumbs() int
	GetCrumbMatrix() (*[][]imgtools.Crumb, error)
}

type CrumbReader interface {
	CrumbPeeker
	ReadCrumb() (c imgtools.Crumb, err error)
}

type CrumbWriter interface {
	CrumbPeeker
	WriteCrumb(c imgtools.Crumb)
	WriteCrumbs(cList []imgtools.Crumb)
}

type CrumbReadWriter interface {
	CrumbReader
	CrumbWriter
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
