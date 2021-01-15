// Copyright © 2020 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package serial

import (
	"fmt"
	"reflect"
	"testing"
)

type MuxTester struct {
	msg    chan []byte
	curbuf []byte
}

func (t *MuxTester) Read(p []byte) (n int, err error) {
	cnt := len(t.curbuf)
	if cnt == 0 {
		fmt.Printf("In mux_test:Read\n")
		t.curbuf = <-t.msg
		fmt.Printf("Out mux_test:Read\n")
		cnt = len(t.curbuf)
	}
	if cnt > len(p) {
		cnt = len(p)
	}
	copy(p, t.curbuf[:cnt])
	t.curbuf = t.curbuf[cnt:]
	return cnt, nil
}

func (t *MuxTester) Write(p []byte) (n int, err error) {
	fmt.Printf("In mux_test:Write\n")
	t.msg <- p
	fmt.Printf("Out mux_test:Write\n")

	return len(p), nil
}

func TestMuxTest(t *testing.T) {
	tester := &MuxTester{msg: make(chan []byte, 10)}
	mux := NewMux(tester)

	for i := 0; i < 10; i++ {
		p1 := fmt.Sprintf("Pass %d this is a test\n", i)
		p2 := fmt.Sprintf("Pass %d If this were real, you'd know!\n", i)

		n, err := mux.Write([]byte(p1))
		if err != nil {
			t.Error(err)
			return
		}
		if n == 0 {
			t.Error("Write returned zero")
			return
		}

		n, err = mux.Write([]byte(p2))
		if err != nil {
			t.Error(err)
			return
		}
		if n == 0 {
			t.Error("Write returned zero")
			return
		}

		buf := make([]byte, 1024)
		n, err = mux.Read(buf)
		if err != nil {
			fmt.Printf("Got error %s from mux.Read pattern 1\n", err)
			t.Error("Data 1 error\n")
		}
		if !reflect.DeepEqual(string(buf[:n]), p1) {
			fmt.Printf("Pass %d data pattern 1 incorrect: got %v expected %v\n",
				i, []byte(buf[:n]), []byte(p1))
			t.Error("Data 1 incorrect\n")
		} else {
			fmt.Printf("Pass %d pattern 1 passes\n", i)
		}

		n, err = mux.Read(buf)
		if err != nil {
			fmt.Printf("Got error %s from mux.Read pattern 2\n", err)
			t.Error("Data 2 error\n")
		}
		if !reflect.DeepEqual(string(buf[:n]), p2) {
			fmt.Printf("Pass %d data pattern 2 incorrect: got %v expected %v\n",
				i, []byte(buf[:n]), []byte(p2))
			t.Error("Data 2 incorrect\n")
		} else {
			fmt.Printf("Pass %d pattern 2 passes\n", i)
		}
	}

	//scanner := bufio.NewScanner(mux)
	//if scanner.Scan() {
	//	fmt.Printf("Final Expected EOF, received %v\n",
	//		scanner.Text())
	//	t.Error("Scanner should have returned EOF but didn't")
	//}
	return
}
