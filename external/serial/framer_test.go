// Copyright © 2020 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package serial

import (
	"fmt"
	"reflect"
	"testing"
)

type FramerTester struct {
}

func (t *FramerTester) Read(p []byte) (n int, err error) {
	p1 := []byte{0x7e, 0x00, 0x01, 0xff, 0x7d, 0x7d, 0x7d, 0x7d,
		0x7d, 0x7e, 0x7d, 0x7e, 0x00, 0x7e}
	copy(p, p1)
	return len(p1), nil
}

func (t *FramerTester) Write(p []byte) (n int, err error) {
	p1 := []byte{0x00, 0x01, 0xff, 0x7d, 0x7d, 0x7d, 0x7d, 0x7d, 0x7e,
		0x7d, 0x7e, 0x00, 0x7e}
	if !reflect.DeepEqual(p, p1) {
		return 0, fmt.Errorf("Data incorrect: %v != %v", p, p1)
	}
	return len(p), nil
}

func TestBasicTest(t *testing.T) {

	tester := &FramerTester{}
	framer := NewFramer(tester)

	p1 := []byte{0x00, 0x01, 0xff, 0x7d, 0x7d, 0x7e, 0x7e, 0x00}

	n, err := framer.Write(p1)
	if err != nil {
		t.Error(err)
		return
	}
	if n == 0 {
		t.Error("Write returned zero")
		return
	}

	buf := make([]byte, 128)
	n, err = framer.Read(buf)
	if err != nil {
		t.Error(err)
		return
	}

	if n == 0 {
		t.Error("Read returned zero")
		return
	}

	buf = buf[0:n]
	if !reflect.DeepEqual(buf, p1) {
		t.Error("Data incorrect:", buf, p1)
	}

	return
}
