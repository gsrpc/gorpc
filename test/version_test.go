package test

import (
	"bytes"
	"fmt"
	"testing"
)

func TestVersionCompatiable(t *testing.T) {
	c0 := NewC0()

	c0.C1 = 1
	c0.C3 = 1.123

	c0.C2.F3 = []byte("hello world")

	var buff bytes.Buffer

	err := WriteC0(&buff, c0)

	if err != nil {
		t.Fatal(err)
	}

	c1, err := ReadC1(&buff)

	if err != nil {
		t.Fatal(err)
	}

	if string(c1.C2.F3) != "hello world" {
		t.Fatal("backward compatiable check failed")
	}

	c1.C2.F4 = []string{"hello", "world"}

	c1.C3 = 2.14

	err = WriteC1(&buff, c1)

	if err != nil {
		t.Fatal(err)
	}

	c0, err = ReadC0(&buff)

	if err != nil {
		t.Fatal(err)
	}

	if c0.C3 != 2.14 {
		fmt.Printf("c3 :%f\n", c0.C3)
		t.Fatal("forward compatiable check failed")
	}

}
