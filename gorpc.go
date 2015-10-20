package gorpc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"

	"github.com/gsdocker/gserrors"
)

//gsbuff public errors
var (
	ErrDecode    = errors.New("decode error")
	ErrWriteNone = errors.New("write nothing")
	ErrEncode    = errors.New("encoding error")
)

//Reader the gsbuff Reader interface, Mixin io.ByteReader and io.Reader interfaces
type Reader interface {
	io.ByteReader
	io.Reader
}

//Writer the gsbuff Writer interface, Mixin io.ByteWriterand io.Writer interfaces
type Writer interface {
	io.ByteWriter
	io.Writer
}

//ReadByte read byte from Reader interface
func ReadByte(reader Reader) (byte, error) {
	val, err := reader.ReadByte()

	if err != nil {
		return 0, gserrors.Newf(err, "read byte error")
	}

	return val, err
}

//ReadSByte read sbyte from Reader interface
func ReadSByte(reader Reader) (int8, error) {
	v, err := reader.ReadByte()
	return int8(v), err
}

//ReadUInt16 read uint16 from Reader interface
func ReadUInt16(reader Reader) (uint16, error) {
	buf := make([]byte, 2)

	_, err := reader.Read(buf)

	if err != nil {
		return 0, gserrors.Newf(err, "read uint16 error")
	}

	return uint16(buf[0]) | uint16(buf[1])<<8, nil
}

//ReadInt16 read uint16 from Reader interface
func ReadInt16(reader Reader) (int16, error) {
	v, err := ReadUInt16(reader)

	return int16(v), err
}

//ReadUInt32 read uint32 from Reader interface
func ReadUInt32(reader Reader) (uint32, error) {
	buf := make([]byte, 4)

	_, err := reader.Read(buf)

	if err != nil {
		return 0, gserrors.Newf(err, "read uint32 error")
	}

	return uint32(buf[3])<<24 | uint32(buf[2])<<16 | uint32(buf[1])<<8 | uint32(buf[0]), nil
}

//ReadInt32 read int32 from Reader interface
func ReadInt32(reader Reader) (int32, error) {
	v, err := ReadUInt32(reader)

	return int32(v), err
}

//ReadUInt64 read uint16 from Reader interface
func ReadUInt64(reader Reader) (uint64, error) {
	buf := make([]byte, 8)
	_, err := reader.Read(buf)

	if err != nil {
		return 0, gserrors.Newf(err, "read uint64 error")
	}

	var ret uint64

	for i, v := range buf {
		ret |= uint64(v) << uint(i*8)
	}

	return ret, nil
}

//ReadInt64 read int16 from Reader interface
func ReadInt64(reader Reader) (int64, error) {
	v, err := ReadUInt64(reader)

	return int64(v), err
}

//ReadFloat32 read float32 from Reader interface
func ReadFloat32(reader Reader) (float32, error) {
	x, err := ReadUInt32(reader)

	if err != nil {
		return 0, err
	}

	return math.Float32frombits(x), nil
}

//ReadFloat64 read float64 from Reader interface
func ReadFloat64(reader Reader) (float64, error) {
	x, err := ReadUInt64(reader)
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(x), nil
}

//ReadString read string
func ReadString(reader Reader) (string, error) {
	len, err := ReadUInt16(reader)

	if err != nil {
		return "", gserrors.Newf(err, "read string len error")
	}

	buff := make([]byte, len)

	_, err = reader.Read(buff)

	if err != nil {
		return "", gserrors.Newf(err, "read string[%d] body error", len)
	}

	return string(buff), nil
}

//ReadBytes read bytes from reader
func ReadBytes(reader Reader, val []byte) error {
	_, err := reader.Read(val)

	if err != nil {
		return gserrors.Newf(err, "read bytes[%d] error", len(val))
	}

	return nil
}

//ReadBool read bool
func ReadBool(reader Reader) (bool, error) {
	b, err := ReadByte(reader)

	if b == 1 {
		return true, err
	}

	return false, err
}

// SkipRead ..
func SkipRead(reader Reader, tag Tag) error {

	switch tag {
	case TagI8:
		_, err := ReadByte(reader)
		return err
	case TagI16:
		_, err := ReadUInt16(reader)
		return err
	case TagI32:
		_, err := ReadUInt32(reader)
		return err
	case TagI64:
		_, err := ReadUInt64(reader)
		return err

	case TagString:
		_, err := ReadString(reader)

		return err

	case TagTable:
		// read fields
		fields, err := ReadByte(reader)
		if err != nil {
			return err
		}

		for i := 0; i < int(fields); i++ {
			tag, err = ReadTag(reader)

			if err != nil {
				return err
			}

			err = SkipRead(reader, tag)

			if err != nil {
				return err
			}
		}

	default:

		if byte(tag)&0xf == byte(TagList) {

			length, err := ReadUInt16(reader)

			if err != nil {
				return err
			}

			tag = Tag((byte(tag) >> 4) & 0x0f)

			for i := 0; i < int(length); i++ {
				err = SkipRead(reader, tag)

				if err != nil {
					return err
				}
			}
		}

		return nil
	}

	return nil
}

//WriteByte write one byte into bytes buffer
func WriteByte(writer Writer, v byte) error {
	return writer.WriteByte(v)
}

//WriteBool write bool byte into bytes buffer
func WriteBool(writer Writer, v bool) error {
	if v {
		return writer.WriteByte(1)
	}
	return writer.WriteByte(0)
}

//WriteSByte write sbyte into stream
func WriteSByte(writer Writer, v int8) error {
	return writer.WriteByte(byte(v))
}

//WriteUInt16 write a little-endian uint16 into a writer stream
func WriteUInt16(writer Writer, v uint16) error {
	if err := WriteByte(writer, byte(v)); err != nil {
		return err
	}
	if err := WriteByte(writer, byte(v>>8)); err != nil {
		return err
	}

	return nil
}

//WriteInt16 write a little-endian int16 into a writer stream
func WriteInt16(writer Writer, v int16) error {
	if err := WriteByte(writer, byte(v)); err != nil {
		return err
	}
	if err := WriteByte(writer, byte(v>>8)); err != nil {
		return err
	}

	return nil
}

//WriteUInt32 write a little-endian uint32 into a writer stream
func WriteUInt32(writer Writer, v uint32) error {
	if err := WriteByte(writer, byte(v)); err != nil {
		return err
	}
	if err := WriteByte(writer, byte(v>>8)); err != nil {
		return err
	}

	if err := WriteByte(writer, byte(v>>16)); err != nil {
		return err
	}
	if err := WriteByte(writer, byte(v>>24)); err != nil {
		return err
	}

	return nil
}

//WriteInt32 write a little-endian int32 into a writer stream
func WriteInt32(writer Writer, v int32) error {
	if err := WriteByte(writer, byte(v)); err != nil {
		return err
	}
	if err := WriteByte(writer, byte(v>>8)); err != nil {
		return err
	}

	if err := WriteByte(writer, byte(v>>16)); err != nil {
		return err
	}
	if err := WriteByte(writer, byte(v>>24)); err != nil {
		return err
	}

	return nil
}

//WriteUInt64 write a little-endian uint64 into a writer stream
func WriteUInt64(writer Writer, v uint64) error {
	for i := uint(0); i < 8; i++ {
		if err := WriteByte(writer, byte(v>>(i*8))); err != nil {
			return err
		}
	}

	return nil
}

//WriteInt64 write a little-endian int64 into a writer stream
func WriteInt64(writer Writer, v int64) error {
	for i := uint(0); i < 8; i++ {
		if err := WriteByte(writer, byte(v>>(i*8))); err != nil {
			return err
		}
	}

	return nil
}

//WriteFloat32 write a little-endian int64 into a writer stream
func WriteFloat32(writer Writer, n float32) error {
	return WriteUInt32(writer, math.Float32bits(n))
}

//WriteFloat64 write a little-endian int64 into a writer stream
func WriteFloat64(writer Writer, n float64) error {
	return WriteUInt64(writer, math.Float64bits(n))
}

//WriteString write string into stream
func WriteString(writer Writer, v string) error {
	err := WriteUInt16(writer, uint16(len(v)))
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte(v))

	return err
}

//WriteBytes write string into stream
func WriteBytes(writer Writer, bytes []byte) error {
	_, err := writer.Write(bytes)
	return err
}

//Read read whole package from stream
func Read(reader io.Reader) ([]byte, error) {
	var header [2]byte

	if _, err := io.ReadFull(reader, header[:]); err != nil {
		return nil, gserrors.Newf(err, "read package length error")
	}

	length := binary.LittleEndian.Uint16(header[:])

	buff := make([]byte, length)

	if _, err := io.ReadFull(reader, buff); err != nil {
		return nil, gserrors.Newf(err, "read package body(%d) error", length)
	}

	return buff, nil
}

// Write write whole input data to stream
func Write(writer io.Writer, buff []byte) (int, error) {
	length := len(buff)

	if length > math.MaxUint16 {
		return 0, gserrors.Newf(ErrEncode, "send package out of range :%d", length)
	}

	var buf [2]byte

	binary.LittleEndian.PutUint16(buf[:], uint16(length))

	length, err := writer.Write(buf[:])

	if err != nil {
		return length, err
	}

	return writer.Write(buff)
}

//Stream the read/write stream
type Stream struct {
	reader io.Reader
	writer io.Writer
	rbuff  *bytes.Buffer
	wbuff  bytes.Buffer
}

// NewStream create new stream
func NewStream(reader io.Reader, writer io.Writer) *Stream {
	return &Stream{
		reader: reader,
		writer: writer,
	}
}

func (stream *Stream) Read(buf []byte) (int, error) {
	if stream.rbuff == nil || stream.rbuff.Len() == 0 {
		cached, err := Read(stream.reader)
		if err != nil {
			return 0, err
		}
		stream.rbuff = bytes.NewBuffer(cached)
	}

	return stream.rbuff.Read(buf)
}

//ReadByte implement Read interface
func (stream *Stream) ReadByte() (byte, error) {
	if stream.rbuff == nil || stream.rbuff.Len() == 0 {
		cached, err := Read(stream.reader)
		if err != nil {
			return 0, err
		}
		stream.rbuff = bytes.NewBuffer(cached)
	}

	return stream.rbuff.ReadByte()
}

func (stream *Stream) Write(buf []byte) (int, error) {
	return stream.wbuff.Write(buf)
}

//WriteByte implement Write interface
func (stream *Stream) WriteByte(val byte) error {
	return stream.wbuff.WriteByte(val)
}

// WBuf .
func (stream *Stream) WBuf() []byte {
	return stream.wbuff.Bytes()
}

//Flush .
func (stream *Stream) Flush() (int, error) {
	len, err := Write(stream.writer, stream.wbuff.Bytes())
	stream.wbuff.Reset()
	return len, err
}
