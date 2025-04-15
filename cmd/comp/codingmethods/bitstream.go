package codingmethods

import (
	"io"

	"github.com/Kagamiin/pixcrumb/cmd/imgtools"
)

type bitstreamMSB struct {
	data          *[]byte
	bytePosition  int
	bitPosition   int
	lengthBits    uint64
	isByteAligned bool
}

func NewBitstreamMSBWriter(data *[]byte) BitstreamMSBWriter {
	return &bitstreamMSB{
		data:          data,
		bytePosition:  0,
		bitPosition:   0,
		lengthBits:    uint64(len(*data)) * 8,
		isByteAligned: false,
	}
}

func NewBitstreamMSBReader(data *[]byte) BitstreamMSBReader {
	return &bitstreamMSB{
		data:          data,
		bytePosition:  0,
		bitPosition:   0,
		lengthBits:    uint64(len(*data)) * 8,
		isByteAligned: false,
	}
}

func (b *bitstreamMSB) GetData() *[]byte {
	return b.data
}

func (b *bitstreamMSB) Reset() {
	b.bytePosition = 0
	b.bitPosition = 0
}

func (b *bitstreamMSB) PeekBit() uint8 {
	return ((*b.data)[b.bytePosition] >> (7 - b.bitPosition)) & 0x01
}

func (b *bitstreamMSB) Tell() int64 {
	return int64(b.bytePosition)*8 + int64(b.bitPosition)
}

func (b *bitstreamMSB) BitsLeft() int64 {
	return int64(b.lengthBits) - int64(b.Tell())
}

func (b *bitstreamMSB) Seek(offset int64, whence int) (int64, error) {
	oldBytePosition := b.bytePosition
	oldBitPosition := b.bitPosition

	switch whence {
	case io.SeekEnd:
		b.bytePosition = int(b.lengthBits) / 8
		b.bitPosition = int(b.lengthBits) % 8
		fallthrough
	case io.SeekCurrent:
		b.bytePosition += int(offset / 8)
		b.bitPosition += int(offset % 8)
		if b.bitPosition > 7 {
			b.bytePosition++
			b.bitPosition -= 8
		}
	case io.SeekStart:
		b.bytePosition = int(offset / 8)
		b.bitPosition = int(offset % 8)
	}

	newPos := b.Tell()
	if newPos < 0 || b.BitsLeft() < 0 {
		b.bytePosition = oldBytePosition
		b.bitPosition = oldBitPosition
		return b.Tell(), io.ErrShortBuffer
	}
	return newPos, nil
}

func (b *bitstreamMSB) ReadBit() (res uint8, err error) {
	if b.bitPosition > 7 {
		b.bitPosition = 0
		b.bytePosition++
	}

	if b.BitsLeft() <= 0 {
		return 0, io.EOF
	}

	res = b.PeekBit()
	b.bitPosition++
	return res, nil
}

func (b *bitstreamMSB) ReadBits(count uint) (res uint64, err error) {
	var bit uint8
	for i := uint(0); i < count; i++ {
		bit, err = b.ReadBit()
		if err != nil {
			return
		}
		res <<= 1
		res |= uint64(bit)
	}
	return
}

func (b *bitstreamMSB) PokeBit(bit uint8) {
	(*b.data)[b.bytePosition] &= ^uint8(0x80) >> b.bitPosition
	(*b.data)[b.bytePosition] |= (bit & 0x01) << (7 - b.bitPosition)
}

func (b *bitstreamMSB) WriteBit(bit uint8) {
	if b.bitPosition > 7 {
		if b.bytePosition >= len(*b.data)-1 {
			*b.data = append(*b.data, uint8(0))
		}
		b.bitPosition = 0
		b.bytePosition++
	} else if len(*b.data) == 0 {
		*b.data = append(*b.data, uint8(0))
	}

	b.PokeBit(bit)
	b.bitPosition++
	b.lengthBits++
}

func (b *bitstreamMSB) WriteBits(val uint64, count uint) {
	for i := uint(1); i <= count; i++ {
		bit := uint8((val >> (count - i)) & 0x01)
		b.WriteBit(bit)
	}
}

func (b *bitstreamMSB) WriteCrumbs(cList []imgtools.Crumb) {
	for _, c := range cList {
		b.WriteBits(uint64(c), 4)
	}
}

func (b *bitstreamMSB) WriteDictEntry(word bitDictWord) {
	b.WriteBits(word.value, word.length)
}

func (b *bitstreamMSB) WriteDictCodedCrumbs(cList []imgtools.Crumb, dict BitDict) {
	for _, c := range cList {
		b.WriteBits(dict[c].value, dict[c].length)
	}
}

func GetNumBitsDictCodedCrumbs(cList []imgtools.Crumb, dict BitDict) (nBits uint64) {
	for _, c := range cList {
		nBits += uint64(dict[c].length)
	}
	return
}

func countBits16(val uint16) uint {
	count := uint(16)
	for count > 0 && val&0x8000 == 0 {
		count--
		val <<= 1
	}
	return count
}

func (b *bitstreamMSB) writeOrder0ExpGolombNumber16(value uint16) {
	if value == 0xFFFF {
		panic("Integer overflow when trying to exp-Golomb encode uint16 value 0xFFFF")
	}
	bitCount := countBits16(value + 1)
	b.WriteBits(0, bitCount-1)
	b.WriteBits(uint64(value+1), bitCount)
}

func (b *bitstreamMSB) WriteOrderKExpGolombNumber16(value uint16, order uint16) {
	b.writeOrder0ExpGolombNumber16(value >> order)
	if order > 0 {
		b.WriteBits(uint64(value&(order-1)), uint(order))
	}
}

func GetNumBitsOrderKExpGolombNumber16(value uint16, order uint16) (nBits uint64) {
	bitCount := countBits16((value >> order) + 1)
	nBits += uint64(bitCount*2 - 1)
	if order > 0 {
		nBits += uint64(order)
	}
	return
}

func (b *bitstreamMSB) ReadOrderKExpGolombNumber16(order uint16) (uint16, error) {
	var leadingBitCount uint16
	bit, err := b.ReadBit()
	if err != nil {
		return 0, err
	}
	for bit == 0 {
		leadingBitCount++
		bit, err = b.ReadBit()
		if err != nil {
			return 0, err
		}
	}
	trailingBitCount := leadingBitCount + order
	suffix, err := b.ReadBits(uint(trailingBitCount))
	return uint16(suffix + (1 << (trailingBitCount)) - 1), err
}
