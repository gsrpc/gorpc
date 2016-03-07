package test

import (
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/gsdocker/gslogger"
	"github.com/gsrpc/gorpc"
	"github.com/gsrpc/gorpc/handler"
)

var G *big.Int

var P *big.Int

var log = gslogger.Get("test")

func init() {
	G, _ = new(big.Int).SetString("6849211231874234332173554215962568648211715948614349192108760170867674332076420634857278025209099493881977517436387566623834457627945222750416199306671083", 0)

	P, _ = new(big.Int).SetString("13196520348498300509170571968898643110806720751219744788129636326922565480984492185368038375211941297871289403061486510064429072584259746910423138674192557", 0)
}

type _MockRESTful struct {
	kv map[string]string //
}

func (mock *_MockRESTful) Put(callSite *gorpc.CallSite, name string, val string) error {
	mock.kv[name] = val

	return nil
}

func (mock *_MockRESTful) Get(callSite *gorpc.CallSite, name string) (string, error) {
	if val, ok := mock.kv[name]; ok {
		return val, nil
	}

	return "", NewResourceError()
}

var stateHandler = handler.NewStateHandler(func(pipeline gorpc.Pipeline, state gorpc.State) {

	pipeline.AddService(MakeRESTfull(1, &_MockRESTful{
		kv: make(map[string]string),
	}))

})

var acceptpor = gorpc.NewAcceptor("test-server",
	gorpc.BuildPipeline(time.Millisecond*10).Handler(
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
	).Handler(
		"state-handler",

		func() gorpc.Handler {
			return stateHandler
		},
	),
)

var clientBuilder = gorpc.NewClientBuilder(
	"test-client",
	gorpc.BuildPipeline(time.Millisecond*10).Handler(

		"dh-client",
		func() gorpc.Handler {
			return handler.NewCryptoClient(gorpc.NewDevice(), handler.NewDHKey(G, P))

		},
	).Handler(

		"heatbeat-client",

		func() gorpc.Handler {

			return handler.NewHeartbeatHandler(5 * time.Second)
		},
	),
)

var upgrader = websocket.Upgrader{} // use default options

func api(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.E("upgrade:", err)
		return
	}

	acceptpor.Accept(fmt.Sprintf("websocket:%s", c.RemoteAddr()), c.UnderlyingConn())
}

func init() {
	gslogger.NewFlags(gslogger.ERROR | gslogger.INFO)
	go gorpc.TCPListen(acceptpor, ":13512")

	http.HandleFunc("/api", api)

	go http.ListenAndServe(":13513", nil)
}

func TestTCPConnect(t *testing.T) {
	client, err := gorpc.TCPConnect(clientBuilder, "tcp-client", "127.0.0.1:13512")

	if err != nil {
		t.Fatal(err)
	}

	api := BindRESTfull(1, client.Pipeline())

	_, err = api.Get(nil, "test")

	if err == nil {
		t.Fatalf("test rpc exception -- failed")
	}

	if _, ok := err.(*ResourceError); !ok {
		t.Fatalf("test rpc exception -- failed:%s", err)
	}

	api.Put(nil, "test", "hello world")

	val, err := api.Get(nil, "test")

	if err != nil {
		t.Fatal(err)
	}

	if val != "hello world" {
		t.Fatalf("test get val error :%s", val)
	}
}

func TestWebSocketConnect(t *testing.T) {
	u, _ := url.Parse("ws://127.0.0.1:13513/api")
	client, err := gorpc.WebSocketConnect(clientBuilder, "websocket-client", u)
	//
	if err != nil {
		t.Fatal(err)
	}

	api := BindRESTfull(1, client.Pipeline())

	_, err = api.Get(nil, "test")

	if err == nil {
		t.Fatalf("test rpc exception -- failed")
	}

	if _, ok := err.(*ResourceError); !ok {
		t.Fatalf("test rpc exception -- failed:%s", err)
	}

	api.Put(nil, "test", "hello world")

	val, err := api.Get(nil, "test")

	if err != nil {
		t.Fatal(err)
	}

	if val != "hello world" {
		t.Fatalf("test get val error :%s", val)
	}
}
