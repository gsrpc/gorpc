package channel

import (
	"com/gsrpc/test"
	"math/big"
	"testing"
	"time"

	"github.com/gsrpc/gorpc"
)

type mockRESTful struct {
	content map[string][]byte
}

func (mock *mockRESTful) Post(name string, content []byte) (err error) {
	mock.content[name] = content
	return nil
}

func (mock *mockRESTful) Get(name string) (retval []byte, err error) {

	val, ok := mock.content[name]

	if ok {
		return val, nil
	}

	return nil, test.NewNotFound()
}

var dispatcher = test.MakeRESTful(0, &mockRESTful{
	content: make(map[string][]byte),
})

func TestConnect(t *testing.T) {

	G, _ := new(big.Int).SetString("6849211231874234332173554215962568648211715948614349192108760170867674332076420634857278025209099493881977517436387566623834457627945222750416199306671083", 0)

	P, _ := new(big.Int).SetString("13196520348498300509170571968898643110806720751219744788129636326922565480984492185368038375211941297871289403061486510064429072584259746910423138674192557", 0)

	exit := make(chan bool)

	go NewTCPServer(
		gorpc.BuildRouter("client").Handler(
			func() gorpc.Handler {
				exit <- true
				return nil
			},
		).Handler(
			func() gorpc.Handler {
				return NewCryptoServer(DHKeyResolve(func(device gorpc.Device) (*DHKey, error) {
					return NewDHKey(G, P), nil
				}))
			},
		),
	).Register(dispatcher).Listen(":13512")

	NewTCPClient(
		"127.0.0.1:13512",
		gorpc.BuildRouter("server").Handler(
			func() gorpc.Handler {
				return NewCryptoClient(G, P)
			},
		).Create(),
	).Connect(time.Second * 2)

	<-exit
}
