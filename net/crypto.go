package net

import (
	"bytes"
	"crypto/cipher"
	"crypto/des"
	"encoding/binary"
	"fmt"
	"math/big"
	"math/rand"
	"time"

	"github.com/gsdocker/gserrors"
	"github.com/gsdocker/gslogger"
	"github.com/gsrpc/gorpc"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

//DHKey 通过Diffie-Hellman算法生成共享密钥
type DHKey struct {
	_G *big.Int //原始根
	_P *big.Int //大素数
	_R *big.Int //随机数
	_E *big.Int //交换密钥
}

//NewDHKey 生成新的DH密钥对象
func NewDHKey(G, P *big.Int) *DHKey {
	key := &DHKey{
		_G: G,
		_P: P,
		_R: big.NewInt(0).Rand(rng, P),
	}

	key._E = big.NewInt(0).Exp(G, key._R, P)

	return key
}

//Exchange 获取交换密钥
func (key *DHKey) Exchange() *big.Int {
	return key._E
}

//Gen 生成共享密钥
func (key *DHKey) Gen(E *big.Int) *big.Int {
	return big.NewInt(0).Exp(E, key._R, key._P)
}

//String 实现ToString接口
func (key *DHKey) String() string {
	return fmt.Sprintf("[DHDHKey]{G:%s,P:%s,R:%s,E:%s}", key._G, key._P, key._R, key._E)
}

// DHKeyResolver .
type DHKeyResolver interface {
	Resolve(device *gorpc.Device) (*DHKey, error)
}

// DHKeyResolve .
type DHKeyResolve func(device *gorpc.Device) (*DHKey, error)

// Resolve .
func (resolve DHKeyResolve) Resolve(device *gorpc.Device) (*DHKey, error) {
	return resolve(device)
}

// PKCS5Padding .
func PKCS5Padding(ciphertext []byte, blockSize int) []byte {

	padding := blockSize - len(ciphertext)%blockSize

	padtext := bytes.Repeat([]byte{byte(padding)}, padding)

	return append(ciphertext, padtext...)
}

// PKCS5UnPadding .
func PKCS5UnPadding(origData []byte) []byte {
	length := len(origData)

	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

// CryptoServer .
type CryptoServer interface {
	GetDevice() *gorpc.Device
}

type _CryptoServer struct {
	gslogger.Log               // Mixin Log APIs
	resovler     DHKeyResolver // resolver
	block        cipher.Block  // cipher block
	device       *gorpc.Device // client device id
}

// NewCryptoServer .
func NewCryptoServer(resovler DHKeyResolver) gorpc.Handler {
	return &_CryptoServer{
		Log:      gslogger.Get("crpyto-server"),
		resovler: resovler,
	}
}

func (handler *_CryptoServer) OpenHandler(context gorpc.Context) error {
	return gorpc.ErrSkip
}

func (handler *_CryptoServer) CloseHandler(context gorpc.Context) {

}

func (handler *_CryptoServer) GetDevice() *gorpc.Device {
	return handler.device
}

func (handler *_CryptoServer) HandleWrite(context gorpc.Context, message *gorpc.Message) (*gorpc.Message, error) {

	if handler.block == nil {

		handler.V("expect WhoAmI message")
		// expect whoAmI message
		if message.Code != gorpc.CodeWhoAmI {
			context.Close()
			return nil, gserrors.Newf(gorpc.ErrRPC, "expect WhoAmI message but got(%s)", message.Code)
		}

		handler.V("parse WhoAmI message")

		whoAmI, err := gorpc.ReadWhoAmI(bytes.NewBuffer(message.Content))

		if err != nil {
			context.Close()
			return nil, err
		}

		val, ok := new(big.Int).SetString(string(whoAmI.Context), 0)

		if !ok {
			context.Close()
			return nil, gserrors.Newf(gorpc.ErrRPC, "parse WhoAmI#Context as big.Int error")
		}

		dhKey, err := handler.resovler.Resolve(whoAmI.ID)

		if err != nil {
			context.Close()
			return nil, err
		}

		message.Code = gorpc.CodeAccept

		message.Content = []byte(dhKey.Exchange().String())

		context.WriteReadPipline(message)

		key := make([]byte, des.BlockSize)

		keyval := dhKey.Gen(val).Uint64()

		binary.BigEndian.PutUint64(key[:8], keyval)

		handler.V("shared key \n\t%d\n\t%v ", keyval, key)

		block, err := des.NewCipher(key)

		if err != nil {
			context.Close()
			return nil, gserrors.Newf(err, "create new des Cipher error")
		}

		handler.block = block

		handler.I("handshake -- success")

		handler.device = whoAmI.ID

		return nil, context.Open()

	}

	if message.Code == gorpc.CodeHeartbeat {
		return message, nil
	}

	blocksize := handler.block.BlockSize()

	if len(message.Content)%blocksize != 0 {
		context.Close()
		return nil, gserrors.Newf(gorpc.ErrRPC, "invalid encrypt data")
	}

	blocks := len(message.Content) / blocksize

	for i := 0; i < blocks; i++ {
		offset := i * blocksize
		v := message.Content[offset : offset+blocksize]
		handler.block.Decrypt(v, v)
	}

	message.Content = PKCS5UnPadding(message.Content)

	return message, nil
}
func (handler *_CryptoServer) HandleRead(context gorpc.Context, message *gorpc.Message) (*gorpc.Message, error) {

	if handler.block != nil {

		blocksize := handler.block.BlockSize()

		content := PKCS5Padding(message.Content, blocksize)

		blocks := len(content) / blocksize

		handler.V("blocksize :%d blocks ：%d", blocksize, blocks)

		for i := 0; i < blocks; i++ {
			offset := i * blocksize
			v := content[offset : offset+blocksize]
			handler.block.Encrypt(v, v)
		}

		message.Content = content
	}

	return message, nil
}

func (handler *_CryptoServer) HandleError(context gorpc.Context, err error) error {
	return err
}

type _CryptoClient struct {
	gslogger.Log               // Mixin Log APIs
	dhKey        *DHKey        // crypto dhkey
	block        cipher.Block  // cipher block
	device       *gorpc.Device // client device id
}

// NewCryptoClient .
func NewCryptoClient(device *gorpc.Device, G, P *big.Int) gorpc.Handler {
	return &_CryptoClient{
		Log:    gslogger.Get("crpyto-client"),
		dhKey:  NewDHKey(G, P),
		device: device,
	}
}

func (handler *_CryptoClient) OpenHandler(context gorpc.Context) error {
	// create whoAmI message

	message := gorpc.NewMessage()

	message.Code = gorpc.CodeWhoAmI

	whoAmI := gorpc.NewWhoAmI()

	whoAmI.ID = handler.device

	whoAmI.Context = []byte(handler.dhKey.Exchange().String())

	var buff bytes.Buffer

	err := gorpc.WriteWhoAmI(&buff, whoAmI)

	if err != nil {
		context.Close()
		return err
	}

	message.Content = buff.Bytes()

	handler.V("send whoAmI handshake")

	context.WriteReadPipline(message)

	return gorpc.ErrSkip

}

func (handler *_CryptoClient) CloseHandler(context gorpc.Context) {

}

func (handler *_CryptoClient) HandleRead(context gorpc.Context, message *gorpc.Message) (*gorpc.Message, error) {
	if handler.block != nil {

		blocksize := handler.block.BlockSize()

		content := PKCS5Padding(message.Content, blocksize)

		blocks := len(content) / blocksize

		for i := 0; i < blocks; i++ {
			offset := i * blocksize
			v := content[offset : offset+blocksize]
			handler.block.Encrypt(v, v)
		}

		message.Content = content

		handler.V("encrypt message content[blocksize :%d, blocks ：%d]", blocksize, blocks)
	}

	return message, nil
}

func (handler *_CryptoClient) HandleWrite(context gorpc.Context, message *gorpc.Message) (*gorpc.Message, error) {

	if handler.block == nil {

		handler.V("expect handshake accept")

		if message.Code != gorpc.CodeAccept {

			context.Close()

			return nil, gserrors.Newf(gorpc.ErrRPC, "expect handshake(Accept) but got(%s)", message.Code)
		}

		handler.V("parse handshake accept")

		val, ok := new(big.Int).SetString(string(message.Content), 0)

		if !ok {
			context.Close()
			return nil, gserrors.Newf(gorpc.ErrRPC, "parse Accept#Content as big.Int error")
		}

		key := make([]byte, des.BlockSize)

		keyval := handler.dhKey.Gen(val).Uint64()

		binary.BigEndian.PutUint64(key[:8], keyval)

		handler.V("shared key \n\t%d\n\t%v ", keyval, key)

		block, err := des.NewCipher(key)

		if err != nil {
			context.Close()
			return nil, gserrors.Newf(err, "create new des Cipher error")
		}

		handler.block = block

		handler.I("handshake -- success")

		return nil, context.Open()
	}

	if message.Code == gorpc.CodeHeartbeat {
		return message, nil
	}

	blocksize := handler.block.BlockSize()

	if len(message.Content)%blocksize != 0 {
		context.Close()
		return nil, gserrors.Newf(gorpc.ErrRPC, "%s invalid encrypt data", context)
	}

	blocks := len(message.Content) / blocksize

	for i := 0; i < blocks; i++ {
		offset := i * blocksize
		v := message.Content[offset : offset+blocksize]
		handler.block.Decrypt(v, v)
	}

	message.Content = PKCS5UnPadding(message.Content)

	return message, nil
}

func (handler *_CryptoClient) HandleError(context gorpc.Context, err error) error {
	return err
}
