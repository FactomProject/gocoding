package gocoding

import (
	"errors"
	"io"
	"math"
	
	"unicode/utf8"
)

func Read(source io.Reader, maxcap int) SliceableRuneReader {
	return &readerRuneReader{
		source: source,
		cbr: &circularRuneBuffer{make([]rune, 0, 16), 0, maxcap},
	}
}

func ReadSlice(slice []rune) SliceableRuneReader {
	return &runeSliceReader{runes: slice}
}

func ReadBytes(slice []byte) SliceableRuneReader {
	return &byteSliceReader{
		runeSliceReader{
			runes: make([]rune, 0, len(slice)),
		},
		slice,
	}
}

func ReadString(data string) SliceableRuneReader {
	return &stringReader{
		runeSliceReader{
			runes: make([]rune, 0, len(data)),
		},
		data,
	}
}

type runeSliceReader struct {
	runes []rune
	cursor int
	mark int
}

func (r *runeSliceReader) Next() rune {
	if r.Done() {
		return EndOfText
	}
	
	r.cursor++
	return r.Peek()
}

func (r *runeSliceReader) Peek() rune {
	if r.cursor == 0 {
		panic("Cannot peek, nothing has been read")
	}
	
	return r.runes[r.cursor-1]
}

func (r *runeSliceReader) Backup() rune {
	if r.cursor == 0 {
		panic(errors.New("Cannot backup past beginning"))
	}
	
	r.cursor--
	return r.Peek()
}

func (r *runeSliceReader) Done() bool {
	return r.cursor == len(r.runes)
}

func (r *runeSliceReader) Mark() {
	if r.cursor == 0 {
		panic("Cannot mark, nothing has been read")
	}
	
	r.mark = r.cursor - 1
}

func (r *runeSliceReader) Slice() SliceableRuneReader {
	return ReadSlice(r.runes[r.mark:r.cursor])
}

func (r *runeSliceReader) String() string {
	return string(r.runes)
}

type byteSliceReader struct {
	runeSliceReader
	remaining []byte
}

func (r *byteSliceReader) Next() rune {
	if r.Done() && r.runeSliceReader.Done() {
		return EndOfText
	}
	
	if !r.runeSliceReader.Done() {
		return r.runeSliceReader.Next()
	}
	
	c, n := utf8.DecodeRune(r.remaining)
	r.remaining = r.remaining[n:]
	
	r.runes = append(r.runes, c)
	r.cursor++
	
	return r.Peek()
}

func (r *byteSliceReader) Peek() rune {
	return r.runeSliceReader.Peek()
}

func (r *byteSliceReader) Done() bool {
	return len(r.remaining) == 0
}

func (r *byteSliceReader) String() string {
	return r.runeSliceReader.String() + string(r.remaining)
}

type stringReader struct {
	runeSliceReader
	remaining string
}

func (r *stringReader) Next() rune {
	if r.Done() {
		return EndOfText
	}
	
	if !r.runeSliceReader.Done() {
		return r.runeSliceReader.Next()
	}
	
	c, n := utf8.DecodeRuneInString(r.remaining)
	r.remaining = r.remaining[n:]
	
	r.runes = append(r.runes, c)
	r.cursor++
	
	return r.Peek()
}

func (r *stringReader) Peek() rune {
	return r.runeSliceReader.Peek()
}

func (r *stringReader) Done() bool {
	return len(r.remaining) == 0
}

func (r *stringReader) String() string {
	return r.runeSliceReader.String() + r.remaining
}

// circularRuneBuffer is a size-limited rune buffer that becomes circular when
// it's capacity reaches a threshold; the methods are written so that the
// caller can pretend the whole buffer exists, but only (up to) maxcap bytes
// from the end are accessible; this is referred to as the imaginary
// non-circular buffer
type circularRuneBuffer struct {
	runes []rune
	offset int32
	maxcap int
}

// retreive the rune at index idx, using circular logic if necessary
func (b *circularRuneBuffer) get(idx int) rune {
	// simple case
	if len(b.runes) < cap(b.runes) {
		return b.runes[idx]
	}
	
	// bounds checking
	if idx < int(b.offset) {
		panic("Index out of bounds: negative index")
	} else if idx >= b.length() {
		panic("Index out of bounds: insufficient length")
	}
	
	// modulo/circular logic
	return b.runes[idx % len(b.runes)]
}

// add a rune to the buffer; will grow until maxcap, then overwrite circularly
func (b *circularRuneBuffer) put(r rune) bool {
	// the slice to grow into, if necessary
	var grow []rune
	
	switch {
	// offset too big, stop
	case b.length() == math.MaxInt32:
		return false
		
	// acting like a normal slice
	case len(b.runes) < cap(b.runes):
		b.runes = b.runes[:len(b.runes)+1]
		b.runes[len(b.runes)-1] = r
		return true
		
	// at max capacity, overwrite circularly
	case cap(b.runes) == b.maxcap:
		idx := int(b.offset) % len(b.runes)
		b.runes[idx] = r
		b.offset++
		return true
		
	// buffer is still small, grow normally
	case cap(b.runes) < b.maxcap / 2:
		grow = make([]rune, len(b.runes), 2*cap(b.runes)+1)
		
	// buffer can be grown, but normal growth would put it past maxcap
	default:
		grow = make([]rune, len(b.runes), b.maxcap)
	}
	
	// copy to the new buffer, reasign the field
	copy(grow, b.runes)
	b.runes = grow
	
	// buffer has been grown, store normally
	return b.put(r)
}

// length of the imaginary non-circular buffer
func (b *circularRuneBuffer) length() int {
	return len(b.runes) + int(b.offset)
}

// get a copy of the slice[i:j] of the imaginary non-circular buffer
func (b *circularRuneBuffer) slice(i, j int) []rune {
	// bounds checking
	if i > j {
		panic("Index out of bounds: slice index i cannot be greater than index j")
	} else if j - i > len(b.runes) {
		panic("Index out of bounds: slice copy length cannot be greater than the original slice length")
	}
	
	// allocate the slice
	slice := make([]rune, j - i)
	
	// simple case
	if len(b.runes) < cap(b.runes) {
		copy(slice, b.runes[i:j])
		return slice
	}
	
	// modulo/circular logic
	i = i % len(b.runes)
	j = j % len(b.runes)
	
	// if the slice doesn't wrap around the bounary, it's still a simple case
	if i <= j {
		copy(slice, b.runes[i:j])
		return slice
	}
	
	// copy the 'first' part
	slice_i := b.runes[i:]
	copy(slice, slice_i)
	
	// copy the 'second' part
	slice_j := b.runes[:j]
	copy(slice[len(slice_i):], slice_j)
	
	return slice
}

// readerRuneReader is a SliceableRuneReader that reads from an io.Reader and
// uses either a sliceRuneBuffer or a circularRuneBuffer for backing up the
// rune stream. The function fields get, put, and slice are initially the
// respective methods of a sliceRuneBuffer. When put returns false, that means
// the sliceRuneBuffer has grown to it's maximum size. At that point, a
// circularRuneBuffer is created, using the sliceRuneBuffer as the rune slice,
// and the fields get, put, and slice are set to the respective methods of the
// circularRuneBuffer
type readerRuneReader struct {
	source io.Reader	// data source
	buffer [64]byte		// buffer to read data in to
	current []byte		// slice to access read data
	
	cbr *circularRuneBuffer // rune buffer
	cursor int			// cursor for the rune buffer
	mark int			// mark/cursor for slicing; see SliceableRuneReader.Mark()
}

func (r *readerRuneReader) Next() rune {
	if r.Done() {
		return EndOfText
	}
	
	var c rune
	var n int
	var err error
	
	// if we're backed up, skip decoding
	if r.cursor < r.cbr.length() {
		goto done
	}
	
	// if there's no data, get some more
	if len(r.current) == 0 {
		// grab and slice the new data
		n, err = r.source.Read(r.buffer[:])
		r.current = r.buffer[:n]
		
		// end of file
		if err == io.EOF {
			r.source = nil
		} else if err != nil {
			panic(err)
		}
	}
	
	// grab the next rune
	for {
		c, n = utf8.DecodeRune(r.current)
		
		if c == utf8.RuneError && r.source != nil {
			// the rune was bad; grab new data, append it to the existing data
			n, err = r.source.Read(r.buffer[:])
			r.current = append(r.current, r.buffer[:n]...)
			
			// end of file
			if err == io.EOF {
				r.source = nil
			} else if err != nil {
				panic(err)
			}
			
			// try again
			continue
		}
		
		// done
		r.current = r.current[n:]
		break
	}
	
	// save the rune
	if !r.cbr.put(c) {
		// the CBR offset has become too large; time to reset
		r.cursor -= int(r.cbr.offset)
		r.mark -= int(r.cbr.offset)
		r.cbr.offset = 0
		
		// save the rune, sanity check
		if !r.cbr.put(c) {
			panic("something is definitely broken")
		}
	}
	
done:
	// get the rune
	r.cursor++
	return r.Peek()
}

func (r *readerRuneReader) Peek() rune {
	if r.cbr.length() == 0 {
		panic("Cannot peek, nothing has been read")
	}
	
	return r.cbr.get(r.cursor-1)
}

func (r *readerRuneReader) Backup() rune {
	if r.cursor == 0 {
		panic(errors.New("Cannot backup past beginning"))
	}
	
	r.cursor--
	return r.Peek()
}

func (r *readerRuneReader) Done() bool {
	if r.cursor < r.cbr.length() {
		return false
	}
	
	return r.source == nil
}

func (r *readerRuneReader) Mark() {
	if r.cursor == 0 {
		panic("Cannot mark, nothing has been read")
	}
	
	r.mark = r.cursor - 1
}

func (r *readerRuneReader) Slice() SliceableRuneReader {
	return ReadSlice(r.cbr.slice(r.mark, r.cursor))
}

func (r *readerRuneReader) String() string {
	panic("readerRuneReader does not support String()")
	return ""
}