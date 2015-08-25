package gorpc

import "fmt"

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

//Code type define -- generate by gsc
type Code byte

//enum Code constants -- generate by gsc
const (
	CodeHeartbeat Code = 0

	CodeWhoAmI Code = 1

	CodeRequest Code = 2

	CodeResponse Code = 3

	CodeAccept Code = 4

	CodeReject Code = 5
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

	case 4:
		return "enum(Code.Accept)"

	case 5:
		return "enum(Code.Reject)"

	}
	return fmt.Sprintf("enum(Unknown(%d))", val)
}

//State type define -- generate by gsc
type State byte

//enum State constants -- generate by gsc
const (
	StateDisconnect State = 0

	StateConnecting State = 1

	StateConnected State = 2

	StateDisconnecting State = 3

	StateClosed State = 4
)

//WriteState write enum to output stream
func WriteState(writer Writer, val State) error {
	return WriteByte(writer, byte(val))
}

//ReadState write enum to output stream
func ReadState(reader Reader) (State, error) {
	val, err := ReadByte(reader)
	return State(val), err
}

//String implement Stringer interface
func (val State) String() string {
	switch val {

	case 0:
		return "enum(State.Disconnect)"

	case 1:
		return "enum(State.Connecting)"

	case 2:
		return "enum(State.Connected)"

	case 3:
		return "enum(State.Disconnecting)"

	case 4:
		return "enum(State.Closed)"

	}
	return fmt.Sprintf("enum(Unknown(%d))", val)
}

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

//OSType type define -- generate by gsc
type OSType byte

//enum OSType constants -- generate by gsc
const (
	OSTypeWindows OSType = 0

	OSTypeLinux OSType = 1

	OSTypeOSX OSType = 2

	OSTypeWP OSType = 3

	OSTypeAndroid OSType = 4

	OSTypeIOS OSType = 5
)

//WriteOSType write enum to output stream
func WriteOSType(writer Writer, val OSType) error {
	return WriteByte(writer, byte(val))
}

//ReadOSType write enum to output stream
func ReadOSType(reader Reader) (OSType, error) {
	val, err := ReadByte(reader)
	return OSType(val), err
}

//String implement Stringer interface
func (val OSType) String() string {
	switch val {

	case 0:
		return "enum(OSType.Windows)"

	case 1:
		return "enum(OSType.Linux)"

	case 2:
		return "enum(OSType.OSX)"

	case 3:
		return "enum(OSType.WP)"

	case 4:
		return "enum(OSType.Android)"

	case 5:
		return "enum(OSType.IOS)"

	}
	return fmt.Sprintf("enum(Unknown(%d))", val)
}

//ArchType type define -- generate by gsc
type ArchType byte

//enum ArchType constants -- generate by gsc
const (
	ArchTypeX86 ArchType = 0

	ArchTypeX64 ArchType = 1

	ArchTypeARM ArchType = 2
)

//WriteArchType write enum to output stream
func WriteArchType(writer Writer, val ArchType) error {
	return WriteByte(writer, byte(val))
}

//ReadArchType write enum to output stream
func ReadArchType(reader Reader) (ArchType, error) {
	val, err := ReadByte(reader)
	return ArchType(val), err
}

//String implement Stringer interface
func (val ArchType) String() string {
	switch val {

	case 0:
		return "enum(ArchType.X86)"

	case 1:
		return "enum(ArchType.X64)"

	case 2:
		return "enum(ArchType.ARM)"

	}
	return fmt.Sprintf("enum(Unknown(%d))", val)
}

//Device -- generate by gsc
type Device struct {
	ID string

	Type string

	Arch ArchType

	OS OSType

	OSVersion string
}

//NewDevice create new struct object with default field val -- generate by gsc
func NewDevice() *Device {
	return &Device{

		ID: "",

		Type: "",

		Arch: ArchTypeX86,

		OS: OSTypeWindows,

		OSVersion: "",
	}
}

//ReadDevice read Device from input stream -- generate by gsc
func ReadDevice(reader Reader) (target *Device, err error) {
	target = NewDevice()

	target.ID, err = ReadString(reader)
	if err != nil {
		return
	}

	target.Type, err = ReadString(reader)
	if err != nil {
		return
	}

	target.Arch, err = ReadArchType(reader)
	if err != nil {
		return
	}

	target.OS, err = ReadOSType(reader)
	if err != nil {
		return
	}

	target.OSVersion, err = ReadString(reader)
	if err != nil {
		return
	}

	return
}

//WriteDevice write Device to output stream -- generate by gsc
func WriteDevice(writer Writer, val *Device) (err error) {

	err = WriteString(writer, val.ID)
	if err != nil {
		return
	}

	err = WriteString(writer, val.Type)
	if err != nil {
		return
	}

	err = WriteArchType(writer, val.Arch)
	if err != nil {
		return
	}

	err = WriteOSType(writer, val.OS)
	if err != nil {
		return
	}

	err = WriteString(writer, val.OSVersion)
	if err != nil {
		return
	}

	return nil
}

//WhoAmI -- generate by gsc
type WhoAmI struct {
	ID *Device

	Context []byte
}

//NewWhoAmI create new struct object with default field val -- generate by gsc
func NewWhoAmI() *WhoAmI {
	return &WhoAmI{

		ID: NewDevice(),

		Context: nil,
	}
}

//ReadWhoAmI read WhoAmI from input stream -- generate by gsc
func ReadWhoAmI(reader Reader) (target *WhoAmI, err error) {
	target = NewWhoAmI()

	target.ID, err = ReadDevice(reader)
	if err != nil {
		return
	}

	target.Context, err = func(reader Reader) ([]byte, error) {
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

//WriteWhoAmI write WhoAmI to output stream -- generate by gsc
func WriteWhoAmI(writer Writer, val *WhoAmI) (err error) {

	err = WriteDevice(writer, val.ID)
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
	}(writer, val.Context)
	if err != nil {
		return
	}

	return nil
}
