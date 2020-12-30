// Copyright © 2020 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package serial

import (
	"bufio"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

type SequencerTester struct {
	msg chan []byte
}

func (t *SequencerTester) Read(p []byte) (n int, err error) {
	p1 := <-t.msg
	copy(p, p1)
	return len(p1), nil
}

func (t *SequencerTester) Write(p []byte) (n int, err error) {
	r := rand.Intn(2)

	switch r {
	case 0:
		fmt.Printf("SequenceTester.Write: Random is send message\n")
		t.msg <- p
	case 1:
		fmt.Printf("SequenceTester.Write: Random is drop message\n")
		break
	case 2:
		fmt.Printf("SequenceTester.Write: Random is duplicate message\n")
		t.msg <- p
		t.msg <- p
	}

	return len(p), nil
}

func TestSequencerTest(t *testing.T) {
	tester := &SequencerTester{msg: make(chan []byte, 10)}
	sequencer := NewSequencer(tester)

	for i := 0; i < 10; i++ {
		fmt.Printf("Pass %d\n", i)
		p1 := fmt.Sprintf("Pass %d this is a test\n", i)
		p2 := fmt.Sprintf("Pass %d If this were real, you'd know!\n", i)

		n, err := sequencer.Write([]byte(p1))
		if err != nil {
			t.Error(err)
			return
		}
		if n == 0 {
			t.Error("Write returned zero")
			return
		}

		n, err = sequencer.Write([]byte(p2))
		if err != nil {
			t.Error(err)
			return
		}
		if n == 0 {
			t.Error("Write returned zero")
			return
		}

		scanner := bufio.NewScanner(sequencer)

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
		if !reflect.DeepEqual(t2, strings.TrimSuffix(string(p2),
			"\n")) {
			fmt.Printf("Pass %d data pattern 2 incorrect: got %v expected %v\n",
				i, t2, string(p2))
			t.Error("Data 2 incorrect\n")
		} else {
			fmt.Printf("Pass %d pattern 2 passes\n", i)
		}
	}

	scanner := bufio.NewScanner(sequencer)
	if scanner.Scan() {
		fmt.Printf("Final Expected EOF, received %v\n",
			scanner.Text())
		t.Error("Scanner should have returned EOF but didn't")
	}
	return
}
