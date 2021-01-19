// Copyright © 2020 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package serial

import (
	"bufio"
	"fmt"
	"reflect"
	"strings"
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
	mux := NewMux(tester, nil)

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

		scanner := bufio.NewScanner(mux)

		if !scanner.Scan() {
			t.Error("Scanner returned premature end")
		}
		t1 := scanner.Text()
		if !reflect.DeepEqual(t1, strings.TrimSuffix(p1, "\n")) {
			fmt.Printf("Pass %d data pattern 1 incorrect: got %v expected %v\n",
				i, t1, p1)
			t.Error("Data 1 incorrect\n")
		} else {
			fmt.Printf("Pass %d pattern 1 passes\n", i)
		}

		if !scanner.Scan() {
			t.Error("Scanner returned premature end")
		}
		t2 := scanner.Text()
		if !reflect.DeepEqual(t2, strings.TrimSuffix(p2, "\n")) {
			fmt.Printf("Pass %d data pattern 2 incorrect: got %v expected %v\n",
				i, t2, p2)
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
