package comp

import (
	"io"

	"github.com/Kagamiin/pixcrumb/cmd/imgtools"
)

type BitstreamBE struct {
	data         *[]byte
	bytePosition int
	bitPosition  int
}

func (b *BitstreamBE) Reset() {
	b.bytePosition = 0
	b.bitPosition = 0
}

func (b *BitstreamBE) PeekBit() uint8 {
	return ((*b.data)[b.bytePosition] >> (7 - b.bitPosition)) & 0x01
}

func (b *BitstreamBE) ReadBit() (res uint8, err error) {
	if b.bitPosition > 7 {
		if b.bytePosition >= len(*b.data)-1 {
			b.bitPosition = 7
			return 0, io.EOF
		}
		b.bitPosition = 0
		b.bytePosition++
	}

	res = b.PeekBit()
	b.bitPosition++
	return res, nil
}

func (b *BitstreamBE) ReadBits(count uint) (res uint64, err error) {
	var bit uint8
	for i := uint(0); i < count; i++ {
		res <<= 1
		bit, err = b.ReadBit()
		if err != nil {
			return
		}
		res |= uint64(bit)
	}
	return
}

func (b *BitstreamBE) PokeBit(bit uint8) {
	(*b.data)[b.bytePosition] &= ^uint8(0x80) >> b.bitPosition
	(*b.data)[b.bytePosition] |= (bit & 0x01) << (7 - b.bitPosition)
}

func (b *BitstreamBE) WriteBit(bit uint8) {
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
}

func (b *BitstreamBE) WriteBits(val uint64, count uint) {
	for i := uint(1); i <= count; i++ {
		bit := uint8((val >> (count - i)) & 0x01)
		b.WriteBit(bit)
	}
}

func (b *BitstreamBE) WriteCrumbs(cList []imgtools.Crumb) {
	for _, c := range cList {
		b.WriteBits(uint64(c), 4)
	}
}

func countBits16(val uint16) uint {
	count := uint(16)
	for count > 0 && val&0x8000 == 0 {
		count--
		val <<= 1
	}
	return count
}

func (b *BitstreamBE) WriteExpGolombNumber16(value uint16) {
	if value == 0xFFFF {
		panic("Integer overflow when trying to exp-Golomb encode uint16 value 0xFFFF")
	}
	bitCount := countBits16(value + 1)
	b.WriteBits(0, bitCount-1)
	b.WriteBits(uint64(value+1), bitCount)
}

func (b *BitstreamBE) WriteExpOrderKGolombNumber16(value uint16, order uint16) {
	b.WriteExpGolombNumber16(value >> order)
	if order > 0 {
		b.WriteBits(uint64(value&(order-1)), uint(order))
	}
}

func (b *BitstreamBE) ReadExpOrderKGolombNumber16(order uint16) (uint16, error) {
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
