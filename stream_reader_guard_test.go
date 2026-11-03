package ole2

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func testPos(sid, size uint32) uint32 { return sid * size }

// A sector chain that loops (0 -> 1 -> 0) used to make Read return data
// forever and, on a chain pointing past the table, called log.Fatal.
func TestStreamReaderCyclicChain(t *testing.T) {
	src := bytes.NewReader(make([]byte, 64))
	r := &StreamReader{sat: []uint32{1, 0}, start: 0, reader: src, offset_of_sector: 0, size_sector: 4, sector_pos: testPos}
	buf := make([]byte, 1000)
	if _, err := r.Read(buf); !errors.Is(err, ErrCorruptChain) {
		t.Fatalf("cyclic chain: err = %v, want ErrCorruptChain", err)
	}
}

func TestStreamReaderChainOutsideTable(t *testing.T) {
	src := bytes.NewReader(make([]byte, 64))
	r := &StreamReader{sat: []uint32{7}, start: 0, reader: src, offset_of_sector: 0, size_sector: 4, sector_pos: testPos}
	buf := make([]byte, 100)
	if _, err := r.Read(buf); !errors.Is(err, ErrCorruptChain) {
		t.Fatalf("out-of-table chain: err = %v, want ErrCorruptChain", err)
	}
}

func TestStreamReaderStopsAtDeclaredSize(t *testing.T) {
	src := bytes.NewReader(make([]byte, 64))
	r := &StreamReader{sat: []uint32{1, 2, 3, ENDOFCHAIN}, start: 0, reader: src, offset_of_sector: 0, size_sector: 4, size: 10, sector_pos: testPos}
	total := 0
	for i := 0; i < 20; i++ {
		n, err := r.Read(make([]byte, 3))
		total += n
		if err == io.EOF {
			break
		}
	}
	if total != 10 {
		t.Fatalf("read %d bytes, want exactly the declared 10", total)
	}
}
