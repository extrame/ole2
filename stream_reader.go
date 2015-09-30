package ole2

import (
	"io"
	"log"
)

var DEBUG = false

type StreamReader struct {
	sat              []uint32
	start            uint32
	ole              *Ole
	offset_of_sector uint32
	offset_in_sector uint32
	size_sector      uint32
	size             int64
	offset           int64
}

func (r *StreamReader) Read(p []byte) (n int, err error) {
	if r.offset_of_sector == ENDOFCHAIN {
		return 0, io.EOF
	}
	for i := 0; i < len(p); i++ {
		if r.offset_in_sector >= r.size_sector {
			r.offset_in_sector = 0
			r.offset_of_sector = r.sat[r.offset_of_sector]
			if r.offset_of_sector == ENDOFCHAIN {
				return i, io.EOF
			}
		}
		pos := r.ole.sector_pos(r.offset_of_sector, r.size_sector) + r.offset_in_sector
		p[i] = r.ole.bts[pos]
		r.offset_in_sector++
		r.offset++
		if r.offset == int64(r.size) {
			return i + 1, io.EOF
		}
	}
	if DEBUG {
		log.Printf("pos:%x,bit:% X", r.offset_of_sector, p)
	}
	return len(p), nil
}

func (r *StreamReader) Seek(offset int64, whence int) (offset_result int64, err error) {

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
		r.offset_of_sector = r.sat[r.offset_of_sector]
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
