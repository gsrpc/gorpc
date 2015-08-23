package channel

import (
	"fmt"
	"math/big"
	"math/rand"
	"time"

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

type _CryptoServer struct {
	dhKey    *DHKey
	resovler DHKeyResolver
}

// DHKeyResolver .
type DHKeyResolver interface {
	Resolve(device gorpc.Device) (*DHKey, error)
}

// DHKeyResolve .
type DHKeyResolve func(device gorpc.Device) (*DHKey, error)

// Resolve .
func (resolve DHKeyResolve) Resolve(device gorpc.Device) (*DHKey, error) {
	return resolve(device)
}

// NewCryptoServer .
func NewCryptoServer(resovler DHKeyResolver) gorpc.Handler {
	return &_CryptoServer{
		resovler: resovler,
	}
}

func (handler *_CryptoServer) HandleSend(router gorpc.Router, message *gorpc.Message) (*gorpc.Message, error) {
	return nil, nil
}

func (handler *_CryptoServer) HandleRecieved(router gorpc.Router, message *gorpc.Message) (*gorpc.Message, error) {
	return nil, nil
}

func (handler *_CryptoServer) HandleClose(router gorpc.Router) {

}

func (handler *_CryptoServer) HandleError(router gorpc.Router, err error) error {
	return err
}

type _CryptoClient struct {
	dhKey *DHKey
}

// NewCryptoClient .
func NewCryptoClient(G, P *big.Int) gorpc.Handler {
	return &_CryptoClient{
		dhKey: NewDHKey(G, P),
	}
}

func (handler *_CryptoClient) HandleSend(router gorpc.Router, message *gorpc.Message) (*gorpc.Message, error) {
	return nil, nil
}

func (handler *_CryptoClient) HandleRecieved(router gorpc.Router, message *gorpc.Message) (*gorpc.Message, error) {
	return nil, nil
}

func (handler *_CryptoClient) HandleClose(router gorpc.Router) {

}

func (handler *_CryptoClient) HandleError(router gorpc.Router, err error) error {
	return err
}
