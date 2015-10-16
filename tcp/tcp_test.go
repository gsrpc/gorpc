package tcp

import (
	"com/gsrpc/test"
	"fmt"
	"math/big"
	"runtime"
	"testing"
	"time"

	"github.com/gsdocker/gslogger"
	"github.com/gsrpc/gorpc"
	"github.com/gsrpc/gorpc/handler"
)

var log = gslogger.Get("profile")

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

var eventLoop = gorpc.NewEventLoop(uint32(runtime.NumCPU()), 2048, 500*time.Millisecond)

var G *big.Int
var P *big.Int

var clientBuilder *ClientBuilder

func init() {

	gslogger.NewFlags(gslogger.ERROR | gslogger.INFO)

	G, _ = new(big.Int).SetString("6849211231874234332173554215962568648211715948614349192108760170867674332076420634857278025209099493881977517436387566623834457627945222750416199306671083", 0)

	P, _ = new(big.Int).SetString("13196520348498300509170571968898643110806720751219744788129636326922565480984492185368038375211941297871289403061486510064429072584259746910423138674192557", 0)

	clientBuilder = BuildClient(
		gorpc.BuildPipeline(eventLoop).Handler(
			"profile",
			gorpc.ProfileHandler,
		).Handler(
			"dh-client",
			func() gorpc.Handler {
				return handler.NewCryptoClient(gorpc.NewDevice(), G, P)
			},
		).Handler(
			"heatbeat-client",
			func() gorpc.Handler {
				return handler.NewHeartbeatHandler(5 * time.Second)
			},
		),
	)

	go NewServer(
		gorpc.BuildPipeline(eventLoop).Handler(
			"profile",
			gorpc.ProfileHandler,
		).Handler(
			"dh-server",
			func() gorpc.Handler {
				return handler.NewCryptoServer(handler.DHKeyResolve(func(device *gorpc.Device) (*handler.DHKey, error) {
					return handler.NewDHKey(G, P), nil
				}))
			},
		).Handler(
			"heatbeat-server",
			func() gorpc.Handler {
				return handler.NewHeartbeatHandler(5 * time.Second)
			},
		),
	).EvtNewPipeline(EvtNewPipeline(func(pipeline gorpc.Pipeline) {
		pipeline.AddService(test.MakeRESTful(0, &mockRESTful{
			content: make(map[string][]byte),
		}))
	})).Listen(":13512")

	for i := 0; i < 10; i++ {
		clientBuilder.Connect(fmt.Sprintf("test-%d", i))
	}

	go func() {
		for _ = range time.Tick(20 * time.Second) {
			log.I("\n%s", gorpc.PrintProfile())
		}
	}()
}

func TestConnect(t *testing.T) {

	client, err := clientBuilder.Connect("test")

	if err != nil {
		t.Fatal(err)
	}

	api := test.BindRESTful(0, client.Pipeline())

	log.D("call Post")

	err = api.Post("nil", nil)

	if err != nil {
		t.Fatal(err)
	}

	log.D("call Post -- success")

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

func BenchmarkPost(t *testing.B) {
	t.StopTimer()
	client, err := clientBuilder.Connect("test")

	if err != nil {
		t.Fatal(err)
	}

	api := test.BindRESTful(0, client.Pipeline())

	t.StartTimer()

	for i := 0; i < t.N; i++ {
		err = api.Post("nil", nil)

		if err != nil {
			t.Fatal(err)
		}
	}
}
