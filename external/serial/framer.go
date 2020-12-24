// Copyright © 2020 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package serial

import (
	"io"
)

type Framer struct {
	c          io.ReadWriter
	readBuffer []byte
	inFrame    bool
	inEscape   bool
}

func NewFramer(c io.ReadWriter) (f *Framer) {
	f = &Framer{c: c}
	return f
}

func (f *Framer) Read(p []byte) (n int, err error) {
	for {
		for len(f.readBuffer) > 0 {
			ch := f.readBuffer[0]
			f.readBuffer = f.readBuffer[1:]
			if !f.inFrame {
				if ch == 0x7e {
					f.inFrame = true
				}
				continue
			}
			if f.inEscape {
				p[n] = ch
				n++
				f.inEscape = false
				continue
			}
			if ch == 0x7d {
				f.inEscape = true
				continue
			}
			if ch != 0x7e {
				p[n] = ch
				n++
				continue
			}
			return
		}
		f.readBuffer = make([]byte, 128)
		nn, err := f.c.Read(f.readBuffer)
		if err != nil || nn == 0 {
			f.inFrame = false
			f.readBuffer = f.readBuffer[:0]
			return 0, err
		}
		f.readBuffer = f.readBuffer[:nn]
	}
}

func (f *Framer) Write(p []byte) (n int, err error) {
	o := make([]byte, 0, 128)

	for _, ch := range p {
		if ch == 0x7d || ch == 0x7e {
			o = append(o, 0x7d, ch)
		} else {
			o = append(o, ch)
		}
	}
	o = append(o, 0x7e)
	for len(o) != 0 {
		nn, err := f.c.Write(o)
		if err != nil {
			return 0, err
		}
		n = n + nn
		o = o[nn:]
	}
	return
}
