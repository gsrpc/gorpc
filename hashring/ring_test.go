package hashring

import (
	"fmt"
	"testing"
)

func TestRing(t *testing.T) {
	auth := New()

	for i := uint32(0); i < 10; i++ {
		auth.Put(fmt.Sprintf("imserver:%d", i), auth)
	}

	_, ok := auth.Get("test")

	if !ok {
		t.Fatal("check get error")
	}

	for i := uint32(0); i < 10; i++ {
		auth.Remove(fmt.Sprintf("imserver:%d", i))
	}

	for i := uint32(0); i < 10; i++ {
		auth.Put(fmt.Sprintf("imserver:%d", i), auth)
	}

	_, ok = auth.Get("test")

	if !ok {
		t.Fatal("check get error")
	}
}
