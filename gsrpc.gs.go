package gorpc

import "fmt"

//Param -- generate by gsc
type Param struct {
	Content []byte
}

//NewParam create new struct object with default field val -- generate by gsc
func NewParam() *Param {
	return &Param{

		Content: nil,
	}
}

//ReadParam read Param from input stream -- generate by gsc
func ReadParam(reader Reader) (target *Param, err error) {
	target = NewParam()

	target.Content, err = func(reader Reader) ([]byte, error) {
		length, err := ReadUInt16(reader)
		if err != nil {
			return nil, err
		}
		if length == 0 {
			return nil, nil
		}
		buff := make([]byte, length)
		err = ReadBytes(reader, buff)
		return buff, err
	}(reader)
	if err != nil {
		return
	}

	return
}

//WriteParam write Param to output stream -- generate by gsc
func WriteParam(writer Writer, val *Param) (err error) {

	err = func(writer Writer, val []byte) error {
		err := WriteUInt16(writer, uint16(len(val)))
		if err != nil {
			return err
		}
		if len(val) != 0 {
			return WriteBytes(writer, val)
		}
		return nil
	}(writer, val.Content)
	if err != nil {
		return
	}

	return nil
}

//Request -- generate by gsc
type Request struct {
	ID uint16

	Method uint16

	Service uint16

	Params []*Param
}

//NewRequest create new struct object with default field val -- generate by gsc
func NewRequest() *Request {
	return &Request{

		ID: uint16(0),

		Method: uint16(0),

		Service: uint16(0),

		Params: nil,
	}
}

//ReadRequest read Request from input stream -- generate by gsc
func ReadRequest(reader Reader) (target *Request, err error) {
	target = NewRequest()

	target.ID, err = ReadUInt16(reader)
	if err != nil {
		return
	}

	target.Method, err = ReadUInt16(reader)
	if err != nil {
		return
	}

	target.Service, err = ReadUInt16(reader)
	if err != nil {
		return
	}

	target.Params, err = func(reader Reader) ([]*Param, error) {
		length, err := ReadUInt16(reader)
		if err != nil {
			return nil, err
		}
		buff := make([]*Param, length)
		for i := uint16(0); i < length; i++ {
			buff[i], err = ReadParam(reader)
			if err != nil {
				return buff, err
			}
		}
		return buff, nil
	}(reader)
	if err != nil {
		return
	}

	return
}

//WriteRequest write Request to output stream -- generate by gsc
func WriteRequest(writer Writer, val *Request) (err error) {

	err = WriteUInt16(writer, val.ID)
	if err != nil {
		return
	}

	err = WriteUInt16(writer, val.Method)
	if err != nil {
		return
	}

	err = WriteUInt16(writer, val.Service)
	if err != nil {
		return
	}

	err = func(writer Writer, val []*Param) error {
		WriteUInt16(writer, uint16(len(val)))
		for _, c := range val {
			err := WriteParam(writer, c)
			if err != nil {
				return err
			}
		}
		return nil
	}(writer, val.Params)
	if err != nil {
		return
	}

	return nil
}

//Response -- generate by gsc
type Response struct {
	ID uint16

	Service uint16

	Exception int8

	Content []byte
}

//NewResponse create new struct object with default field val -- generate by gsc
func NewResponse() *Response {
	return &Response{

		ID: uint16(0),

		Service: uint16(0),

		Exception: int8(0),

		Content: nil,
	}
}

//ReadResponse read Response from input stream -- generate by gsc
func ReadResponse(reader Reader) (target *Response, err error) {
	target = NewResponse()

	target.ID, err = ReadUInt16(reader)
	if err != nil {
		return
	}

	target.Service, err = ReadUInt16(reader)
	if err != nil {
		return
	}

	target.Exception, err = ReadSByte(reader)
	if err != nil {
		return
	}

	target.Content, err = func(reader Reader) ([]byte, error) {
		length, err := ReadUInt16(reader)
		if err != nil {
			return nil, err
		}
		if length == 0 {
			return nil, nil
		}
		buff := make([]byte, length)
		err = ReadBytes(reader, buff)
		return buff, err
	}(reader)
	if err != nil {
		return
	}

	return
}

//WriteResponse write Response to output stream -- generate by gsc
func WriteResponse(writer Writer, val *Response) (err error) {

	err = WriteUInt16(writer, val.ID)
	if err != nil {
		return
	}

	err = WriteUInt16(writer, val.Service)
	if err != nil {
		return
	}

	err = WriteSByte(writer, val.Exception)
	if err != nil {
		return
	}

	err = func(writer Writer, val []byte) error {
		err := WriteUInt16(writer, uint16(len(val)))
		if err != nil {
			return err
		}
		if len(val) != 0 {
			return WriteBytes(writer, val)
		}
		return nil
	}(writer, val.Content)
	if err != nil {
		return
	}

	return nil
}

//Code type define -- generate by gsc
type Code byte

//enum Code constants -- generate by gsc
const (
	CodeHeartbeat Code = 0

	CodeWhoAmI Code = 1

	CodeRequest Code = 2

	CodeResponse Code = 3
)

//WriteCode write enum to output stream
func WriteCode(writer Writer, val Code) error {
	return WriteByte(writer, byte(val))
}

//ReadCode write enum to output stream
func ReadCode(reader Reader) (Code, error) {
	val, err := ReadByte(reader)
	return Code(val), err
}

//String implement Stringer interface
func (val Code) String() string {
	switch val {

	case 0:
		return "enum(Code.Heartbeat)"

	case 1:
		return "enum(Code.WhoAmI)"

	case 2:
		return "enum(Code.Request)"

	case 3:
		return "enum(Code.Response)"

	}
	return fmt.Sprintf("enum(Unknown(%d))", val)
}

//Message -- generate by gsc
type Message struct {
	Code Code

	Content []byte
}

//NewMessage create new struct object with default field val -- generate by gsc
func NewMessage() *Message {
	return &Message{

		Code: CodeHeartbeat,

		Content: nil,
	}
}

//ReadMessage read Message from input stream -- generate by gsc
func ReadMessage(reader Reader) (target *Message, err error) {
	target = NewMessage()

	target.Code, err = ReadCode(reader)
	if err != nil {
		return
	}

	target.Content, err = func(reader Reader) ([]byte, error) {
		length, err := ReadUInt16(reader)
		if err != nil {
			return nil, err
		}
		if length == 0 {
			return nil, nil
		}
		buff := make([]byte, length)
		err = ReadBytes(reader, buff)
		return buff, err
	}(reader)
	if err != nil {
		return
	}

	return
}

//WriteMessage write Message to output stream -- generate by gsc
func WriteMessage(writer Writer, val *Message) (err error) {

	err = WriteCode(writer, val.Code)
	if err != nil {
		return
	}

	err = func(writer Writer, val []byte) error {
		err := WriteUInt16(writer, uint16(len(val)))
		if err != nil {
			return err
		}
		if len(val) != 0 {
			return WriteBytes(writer, val)
		}
		return nil
	}(writer, val.Content)
	if err != nil {
		return
	}

	return nil
}
