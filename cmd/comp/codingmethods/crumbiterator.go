package codingmethods

import (
	"errors"
	"fmt"
	"io"

	"github.com/Kagamiin/pixcrumb/cmd/imgtools"
)

type crumbIterator struct {
	mtx          *[][]imgtools.Crumb
	index        int64
	totalDataLen uint64
	width        uint64
}

var (
	ErrCrumbIndexOutOfBounds        = errors.New("crumb index out of bounds")
	ErrCrumbDataNotAlignedToMatrix  = errors.New("crumb data does not fill the last line of the matrix")
	ErrCrumbMatrixWidthInconsistent = errors.New("crumb matrix has inconsistent line widths")
)

func checkCrumbMatrixConsistency(mtx *[][]imgtools.Crumb) error {
	if len(*mtx) == 0 {
		return nil
	}
	expectedWidth := len((*mtx)[0])
	for _, row := range *mtx {
		if len(row) != expectedWidth {
			return ErrCrumbDataNotAlignedToMatrix
		}
	}
	return nil
}

func NewCrumbReader(mtx *[][]imgtools.Crumb) (CrumbReader, error) {
	checkCrumbMatrixConsistency(mtx)
	return &crumbIterator{
		mtx:          mtx,
		index:        0,
		totalDataLen: uint64(len(*mtx) * len((*mtx)[0])),
		width:        uint64(len((*mtx)[0])),
	}, nil
}

func NewCrumbWriter(width uint64) CrumbWriter {
	data := make([][]imgtools.Crumb, 0)
	return &crumbIterator{
		mtx:          &data,
		index:        0,
		totalDataLen: 0,
		width:        width,
	}
}

func (ci *crumbIterator) Length() uint64 {
	return ci.totalDataLen
}

func (ci *crumbIterator) Seek(offset int64, whence int) (int64, error) {
	oldSeek := ci.index
	switch whence {
	case io.SeekCurrent:
		ci.index += offset
	case io.SeekStart:
		ci.index = offset
	case io.SeekEnd:
		ci.index = int64(ci.totalDataLen) + offset
	}
	if ci.index > int64(ci.totalDataLen) || ci.index < 0 {
		ci.index = oldSeek
		return oldSeek, io.ErrShortBuffer
	}
	return ci.index, nil
}

func (ci *crumbIterator) Tell() int64 {
	return ci.index
}

func (ci *crumbIterator) linearToMortonIndex(idx int64) (yPos int, xPos int) {
	yPos = int(idx / int64(ci.width))
	xOffs := int(idx % int64(ci.width))
	if yPos&1 == 0 {
		xPos = xOffs
	} else {
		xPos = int(ci.width) - xOffs - 1
	}
	return
}

func (ci *crumbIterator) GetHeightCrumbs() int {
	return len(*ci.mtx)
}

func (ci *crumbIterator) IsLengthAligned() bool {
	_, x := ci.linearToMortonIndex(int64(ci.totalDataLen))
	return x == 0
}

func (ci *crumbIterator) IsAtEnd() bool {
	return ci.index >= int64(ci.totalDataLen)
}

func (ci *crumbIterator) GetCrumbMatrix() (*[][]imgtools.Crumb, error) {
	if !ci.IsLengthAligned() {
		return nil, ErrCrumbDataNotAlignedToMatrix
	}
	return ci.mtx, nil
}

func (ci *crumbIterator) PeekCrumbAt(offset int64, relative bool) (imgtools.Crumb, error) {
	var index int64
	if relative {
		index = ci.index + offset
	} else {
		index = offset
	}
	if index >= int64(ci.totalDataLen) {
		return 0, fmt.Errorf(
			"%w (tried to access index %d where crumb data has length %d (size %dx%d ))",
			ErrCrumbIndexOutOfBounds,
			index,
			ci.totalDataLen,
			ci.width,
			len(*ci.mtx),
		)
	}
	if index < 0 {
		return 0, fmt.Errorf("%w (tried to access negative index %d)", ErrCrumbIndexOutOfBounds, index)
	}
	y, x := ci.linearToMortonIndex(index)
	return (*ci.mtx)[y][x], nil
}

func (ci *crumbIterator) PeekCrumb() (c imgtools.Crumb, err error) {
	return ci.PeekCrumbAt(ci.index, false)
}

func (ci *crumbIterator) PeekNCrumbsAt(n uint64, offset int64, relative bool) (cList []imgtools.Crumb, err error) {
	if _, err = ci.PeekCrumbAt(offset, relative); errors.Is(err, ErrCrumbIndexOutOfBounds) {
		return nil, err
	}
	for idx := range int64(n) {
		var c imgtools.Crumb
		c, err = ci.PeekCrumbAt(offset+idx, relative)
		if errors.Is(err, ErrCrumbIndexOutOfBounds) {
			return cList, io.ErrUnexpectedEOF
		}
		cList = append(cList, c)
	}
	return
}

func (ci *crumbIterator) PeekNCrumbs(n uint64) (cList []imgtools.Crumb, err error) {
	return ci.PeekNCrumbsAt(n, 0, true)
}

func (ci *crumbIterator) ReadCrumb() (c imgtools.Crumb, err error) {
	if ci.index >= int64(ci.totalDataLen) {
		return 0, io.EOF
	}
	c, err = ci.PeekCrumb()
	ci.index++
	return
}

func (ci *crumbIterator) ReadCrumbs(n int) (cList []imgtools.Crumb, err error) {
	for range n {
		c, readErr := ci.ReadCrumb()
		if errors.Is(readErr, io.EOF) {
			return cList, io.ErrUnexpectedEOF
		}
		cList = append(cList, c)
	}
	return
}

func (ci *crumbIterator) WriteCrumb(c imgtools.Crumb) {
	y, x := ci.linearToMortonIndex(ci.index)
	if y > len(*ci.mtx)-1 {
		*ci.mtx = append(*ci.mtx, make([]imgtools.Crumb, ci.width))
	}
	(*ci.mtx)[y][x] = c
	ci.index++
}

func (ci *crumbIterator) WriteCrumbs(cList []imgtools.Crumb) {
	for _, c := range cList {
		ci.WriteCrumb(c)
	}
}
