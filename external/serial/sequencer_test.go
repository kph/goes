// Copyright © 2020 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package serial

import (
	"fmt"
	"reflect"
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
	t.msg <- p
	t.msg <- p

	return len(p), nil
}

func TestSequencerTest(t *testing.T) {
	tester := &SequencerTester{msg: make(chan []byte, 10)}
	sequencer := NewSequencer(tester)

	for i := 0; i < 10; i++ {
		fmt.Printf("Pass %d\n", i)
		p1 := []byte("This is a test")
		p2 := []byte("If this were real, you'd know!")

		n, err := sequencer.Write(p1)
		if err != nil {
			t.Error(err)
			return
		}
		if n == 0 {
			t.Error("Write returned zero")
			return
		}

		n, err = sequencer.Write(p2)
		if err != nil {
			t.Error(err)
			return
		}
		if n == 0 {
			t.Error("Write returned zero")
			return
		}

		buf := make([]byte, 128)
		n, err = sequencer.Read(buf)
		if err != nil {
			t.Error(err)
			return
		}

		if n == 0 {
			t.Error("Read returned zero")
			return
		}

		if !reflect.DeepEqual(buf[:n], p1) {
			t.Error("Data incorrect:", buf[:n], p1)
		}

		n, err = sequencer.Read(buf)
		if err != nil {
			t.Error(err)
			return
		}

		if n == 0 {
			t.Error("Read returned zero")
			return
		}

		if !reflect.DeepEqual(buf[:n], p2) {
			t.Error("Data incorrect:", buf[:n], p2)
		}
	}
	return
}
