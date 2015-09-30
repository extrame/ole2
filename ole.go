package ole2

import (
	"encoding/binary"
	"io"
)

var ENDOFCHAIN = uint32(0xFFFFFFFE) //-2
var FREESECT = uint32(0xFFFFFFFF)   // -1

type Ole struct {
	header   *Header
	Lsector  uint32
	Lssector uint32
	SecID    []uint32
	SSecID   []uint32
	Files    []File
	bts      []byte
}

func Open(bts []byte, charset string) (ole *Ole, err error) {
	var header *Header
	if header, err = parseHeader(bts[:512]); err == nil {
		ole = new(Ole)
		ole.bts = bts
		ole.header = header
		ole.Lsector = 512 //TODO
		ole.Lssector = 64 //TODO
		ole.bts = bts
		ole.readMSAT()
		return ole, nil
	}

	return nil, err
}

func (o *Ole) ListDir() (dir []*File, err error) {
	sector := o.stream_read(o.header.Dirstart, 0)
	dir = make([]*File, 0)
	for {
		d := new(File)
		err = binary.Read(sector, binary.LittleEndian, d)
		if err == nil && d.Type != EMPTY {
			dir = append(dir, d)
		} else {
			break
		}
	}
	if err == io.EOF && dir != nil {
		return dir, nil
	}

	return
}

func (o *Ole) OpenFile(file *File) io.ReadSeeker {
	if file.Size < o.header.Sectorcutoff {
		return o.short_stream_read(file.Sstart, file.Size)
	} else {
		return o.stream_read(file.Sstart, file.Size)
	}
}

// Read MSAT
func (o *Ole) readMSAT() {
	// int sectorNum;

	count := uint32(109)
	if o.header.Cfat < 109 {
		count = o.header.Cfat
	}

	for i := uint32(0); i < count; i++ {
		sector := o.sector_read(o.header.Msat[i])
		sids := sector.AllValues(o.Lsector)

		o.SecID = append(o.SecID, sids...)
	}

	for sid := o.header.Difstart; sid != ENDOFCHAIN; {
		sector := o.sector_read(sid)
		sids := sector.MsatValues(o.Lsector)

		for _, sid := range sids {
			sector := o.sector_read(sid)
			sids := sector.AllValues(o.Lsector)

			o.SecID = append(o.SecID, sids...)
		}

		sid = sector.NextSid(o.Lsector)
	}

	for i := uint32(0); i < o.header.Csfat; i++ {
		sid := o.header.Sfatstart

		if sid != ENDOFCHAIN {
			sector := o.sector_read(sid)

			sids := sector.MsatValues(o.Lsector)

			o.SSecID = append(o.SSecID, sids...)

			sid = sector.NextSid(o.Lsector)
		}
	}

}

func (o *Ole) stream_read(sid uint32, size uint32) *StreamReader {
	return &StreamReader{o.SecID, sid, o, sid, 0, o.Lsector, int64(size), 0}
}

func (o *Ole) short_stream_read(sid uint32, size uint32) *StreamReader {
	return &StreamReader{o.SSecID, sid, o, sid, 0, o.Lssector, int64(size), 0}
}

func (o *Ole) sector_read(sid uint32) Sector {
	pos := o.sector_pos(sid, o.Lsector)
	bts := o.bts[pos : pos+o.Lsector]
	return Sector(bts)
}

func (o *Ole) short_sector_read(sid uint32) Sector {
	pos := o.sector_pos(sid, o.Lssector)
	return Sector(o.bts[pos : pos+o.Lssector])
}

func (o *Ole) sector_pos(sid uint32, size uint32) uint32 {
	return 512 + sid*size
}
