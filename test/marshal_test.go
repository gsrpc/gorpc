package test

import (
	"bytes"
	"com/gsrpc/test"
	"testing"

	"github.com/gsrpc/gorpc"
)

var request = gorpc.NewRequest()

func init() {

	var buff bytes.Buffer

	test.WriteKV(&buff, test.NewKV())

	request.Params = append(request.Params, &gorpc.Param{Content: buff.Bytes()})

	request.Params = append(request.Params, &gorpc.Param{Content: buff.Bytes()})

	request.Params = append(request.Params, &gorpc.Param{Content: buff.Bytes()})
}

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

func TestMarshal(t *testing.T) {
	var buff bytes.Buffer

	err := gorpc.WriteRequest(&buff, request)

	if err != nil {
		t.Fatal(err)
	}

	requestNew, err := gorpc.ReadRequest(&buff)

	if err != nil {
		t.Fatal(err)
	}

	if len(requestNew.Params) != 3 {
		t.Fatal("unmarshal error")
	}
}

func TestRPC(t *testing.T) {
	dispatcher := test.MakeRESTful(0, &mockRESTful{
		content: make(map[string][]byte),
	})

	api := test.BindRESTful(0, gorpc.Send(func(call *gorpc.Request) (gorpc.Future, error) {

		return gorpc.Wait(func() (callReturn *gorpc.Response, err error) {
			return dispatcher.Dispatch(call)
		}), nil

	}))

	api.Post("nil", nil)

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

func BenchmarkMarshal(b *testing.B) {
	var buff bytes.Buffer

	for i := 0; i < b.N; i++ {
		err := gorpc.WriteRequest(&buff, request)

		if err != nil {
			b.Fatal(err)
		}

		_, err = gorpc.ReadRequest(&buff)

		if err != nil {
			b.Fatal(err)
		}
	}
}
