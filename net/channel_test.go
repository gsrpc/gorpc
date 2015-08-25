package net

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

	serverSink := gorpc.NewSink("server-sink", time.Second*5, 1024)

	serverSink.Register(dispatcher)

	go NewTCPServer(
		gorpc.BuildPipeline().Handler(
			"log-server",
			func() gorpc.Handler {
				return gorpc.LoggerHandler()
			},
		).Handler(
			"dh-server",
			func() gorpc.Handler {
				return NewCryptoServer(DHKeyResolve(func(device *gorpc.Device) (*DHKey, error) {
					return NewDHKey(G, P), nil
				}))
			},
		).Handler(
			"sink-server",
			func() gorpc.Handler {
				return serverSink
			},
		),
	).Listen(":13512")

	clientSink := gorpc.NewSink("client-sink", time.Second*5, 1024)

	client := NewTCPClient(
		"127.0.0.1:13512",
		gorpc.BuildPipeline().Handler(
			"log-client",
			func() gorpc.Handler {
				return gorpc.LoggerHandler()
			},
		).Handler(
			"dh-client",
			func() gorpc.Handler {
				return NewCryptoClient(gorpc.NewDevice(), G, P)
			},
		).Handler(
			"sink-client",
			func() gorpc.Handler {
				return clientSink
			},
		),
	).Connect(0)

	api := test.BindRESTful(0, clientSink)

	client.D("call Post")

	err := api.Post("nil", nil)

	if err != nil {
		t.Fatal(err)
	}

	client.D("call Post -- success")

	content, err := api.Get("nil")

	if err != nil {
		t.Fatal(err)
	}

	if content != nil {
		t.Fatal("rpc test error")
	}

	api.Post("hello", []byte("hello world"))

	content, err = api.Get("hello")

	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "hello world" {
		t.Fatal("rpc test error")
	}

	_, err = api.Get("hello2")

	if err == nil {
		t.Fatal("expect (*test.NotFound)exception")
	}

	if _, ok := err.(*test.NotFound); !ok {
		t.Fatal("expect (*test.NotFound)exception")
	}
}
