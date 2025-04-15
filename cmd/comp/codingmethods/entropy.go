package codingmethods

import "github.com/Kagamiin/pixcrumb/cmd/imgtools"

var crumbHistogram = [16]uint64{190717, 25529, 32942, 16299, 28947, 35376, 18160, 19100, 29189, 17495, 54283, 20529, 17301, 18498, 19300, 93882}

type bitDictWord struct {
	value  uint64
	length uint
}

type BitDict map[imgtools.Crumb]bitDictWord

const (
	TOKEN_END_OF_LITERALS imgtools.Crumb = 16
)

var DictRLE = map[imgtools.Crumb]bitDictWord{
	0x0: {0b00, 2},
	0xF: {0b01, 2},
	0xA: {0b100, 3},
	0x5: {0b101, 3},
	0x2: {0b1100, 4},
	0x8: {0b1101, 4},
	0x4: {0b11100, 5},
	0x1: {0b11101, 5},
	0xB: {0b1111000, 7},
	0xE: {0b1111001, 7},
	0x7: {0b1111010, 7},
	0xD: {0b1111011, 7},
	0x6: {0b1111100, 7},
	0x9: {0b1111101, 7},
	0xC: {0b1111110, 7},
	0x3: {0b1111111, 7},
}

var DictLZ = map[imgtools.Crumb]bitDictWord{
	TOKEN_END_OF_LITERALS: {0b00, 2},
	0x0:                   {0b01, 2},
	0xF:                   {0b10, 2},
	0xA:                   {0b1100, 4},
	0x5:                   {0b1101, 4},
	0x2:                   {0b11100, 5},
	0x8:                   {0b11101, 5},
	0x4:                   {0b111100, 6},
	0x1:                   {0b111101, 6},
	0xB:                   {0b11111000, 8},
	0xE:                   {0b11111001, 8},
	0x7:                   {0b11111010, 8},
	0xD:                   {0b11111011, 8},
	0x6:                   {0b11111100, 8},
	0x9:                   {0b11111101, 8},
	0xC:                   {0b11111110, 8},
	0x3:                   {0b11111111, 8},
}
