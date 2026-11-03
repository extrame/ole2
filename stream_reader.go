package ole2

import (
	"errors"
	"io"
	"log"
)

var DEBUG = false

// ErrCorruptChain is returned when a sector chain is cyclic or points outside
// the allocation table, which only happens in a damaged file.
var ErrCorruptChain = errors.New("ole2: corrupt sector chain")

type StreamReader struct {
	sat              []uint32
	start            uint32
	reader           io.ReadSeeker
	offset_of_sector uint32
	offset_in_sector uint32
	size_sector      uint32
	size             int64
	offset           int64
	sector_pos       func(uint32, uint32) uint32
	chain_steps      int // sectors followed in the current Read/Seek, bounds cyclic chains
}

// follow moves to the next sector of the chain, failing instead of looping
// forever or indexing outside the allocation table.
func (r *StreamReader) follow() error {
	r.chain_steps++
	if r.chain_steps > len(r.sat)+1 || r.offset_of_sector >= uint32(len(r.sat)) {
		return ErrCorruptChain
	}
	r.offset_of_sector = r.sat[r.offset_of_sector]
	return nil
}

func (r *StreamReader) Read(p []byte) (n int, err error) {
	r.chain_steps = 0
	if r.offset_of_sector == ENDOFCHAIN {
		return 0, io.EOF
	}
	// A stream with a known size never yields more than that many bytes; this
	// also stops a cyclic chain from feeding the caller data forever.
	if r.size > 0 {
		remaining := r.size - r.offset
		if remaining <= 0 {
			return 0, io.EOF
		}
		if int64(len(p)) > remaining {
			p = p[:remaining]
		}
	}
	defer func() { r.offset += int64(n) }()
	pos := r.sector_pos(r.offset_of_sector, r.size_sector) + r.offset_in_sector
	r.reader.Seek(int64(pos), 0)
	readed := uint32(0)
	for remainLen := uint32(len(p)) - readed; remainLen > r.size_sector-r.offset_in_sector; remainLen = uint32(len(p)) - readed {
		if n, err := r.reader.Read(p[readed : readed+r.size_sector-r.offset_in_sector]); err != nil {
			return int(readed) + n, err
		} else {
			readed += uint32(n)
			r.offset_in_sector = 0
			if err := r.follow(); err != nil {
				return int(readed), err
			}
			if r.offset_of_sector == ENDOFCHAIN {
				return int(readed), io.EOF
			}
			pos := r.sector_pos(r.offset_of_sector, r.size_sector) + r.offset_in_sector
			r.reader.Seek(int64(pos), 0)
		}
	}
	if n, err := r.reader.Read(p[readed:len(p)]); err == nil {
		r.offset_in_sector += uint32(n)
		if DEBUG {
			log.Printf("pos:%x,bit:% X", r.offset_of_sector, p)
		}
		return len(p), nil
	} else {
		return int(readed) + n, err
	}

}

func (r *StreamReader) Seek(offset int64, whence int) (offset_result int64, err error) {
	r.chain_steps = 0

	if whence == 0 {
		r.offset_of_sector = r.start
		r.offset_in_sector = 0
		r.offset = offset
	} else {
		r.offset += offset
	}

	if r.offset_of_sector == ENDOFCHAIN {
		return r.offset, io.EOF
	}

	for offset >= int64(r.size_sector-r.offset_in_sector) {
		if err = r.follow(); err != nil {
			return r.offset, err
		}
		offset -= int64(r.size_sector - r.offset_in_sector)
		r.offset_in_sector = 0
		if r.offset_of_sector == ENDOFCHAIN {
			err = io.EOF
			goto return_res
		}
	}

	if r.size <= r.offset {
		err = io.EOF
		r.offset = r.size
	} else {
		r.offset_in_sector += uint32(offset)
	}
return_res:
	offset_result = r.offset
	return
}
