package parser

import (
	"encoding/binary"
	"io"
)

type FileParser struct {
	r      io.ReadSeeker
	endian binary.ByteOrder
	err    error
}

func (p *FileParser) Read(data any) {
	if p.err != nil {
		return
	}
	p.err = binary.Read(p.r, p.endian, data)
}

func (p *FileParser) ReadBytes(n int) []byte {
	if p.err != nil {
		return nil
	}
	buf := make([]byte, n)
	_, p.err = io.ReadFull(p.r, buf)
	return buf
}

func (p *FileParser) Skip(n int) {
	if p.err != nil {
		return
	}
	_, p.err = p.r.Seek(int64(n), io.SeekCurrent)
}

func (p *FileParser) Seek(offset uint64) {
	if p.err != nil {
		return
	}
	_, p.err = p.r.Seek(int64(offset), io.SeekStart)
}

func (p *FileParser) ReadFloat32() float32 {
	var f float32
	p.Read(&f)
	return f
}

func (p *FileParser) ReadFloat64() float64 {
	var f float64
	p.Read(&f)
	return f
}

func (p *FileParser) ReadUInt8() uint8 {
	var u uint8
	p.Read(&u)
	return u
}

func (p *FileParser) ReadUInt16() uint16 {
	var u uint16
	p.Read(&u)
	return u
}

func (p *FileParser) ReadUInt32() uint32 {
	var u uint32
	p.Read(&u)
	return u
}

func (p *FileParser) ReadUInt64() uint64 {
	var u uint64
	p.Read(&u)
	return u
}

func (p *FileParser) ReadString() string {
	var length uint32
	p.Read(&length)
	return string(p.ReadBytes(int(length)))
}

func (p *FileParser) ReadBool() bool {
	var b uint8
	p.Read(&b)
	return b != 0
}

func (p *FileParser) ReadLine() string {
	if p.err != nil {
		return ""
	}
	var line []byte
	buf := make([]byte, 1)
	for {
		_, p.err = p.r.Read(buf)
		if p.err != nil {
			return ""
		}
		if buf[0] == '\n' {
			break
		}
		line = append(line, buf[0])
	}
	return string(line)
}
