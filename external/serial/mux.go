// Copyright © 2020 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package serial

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
)

type Mux struct {
	c        io.ReadWriter
	databuf  []byte
	recvCond *sync.Cond
	listener func(stream int) error
}

type Stream struct {
}

func NewMux(c io.ReadWriter, listener func(stream int) (err error)) (m *Mux) {
	m = &Mux{
		c:        c,
		listener: listener,
		recvCond: sync.NewCond(&sync.Mutex{}),
	}
	go m.backgroundRead()
	return m
}

func (m *Mux) backgroundRead() (err error) {
	var pktlen uint16
	var ctrl byte
	var stream byte
	var databuf []byte

	for {
		hdr := make([]byte, 4)
		for hdrlen := 0; hdrlen < 4; {
			nn, err := m.c.Read(hdr[hdrlen:])
			if err != nil {
				return fmt.Errorf("mux: Error reading header: %w",
					err)
			}
			hdrlen += nn
		}

		buf := bytes.NewReader(hdr)
		err = binary.Read(buf, binary.LittleEndian, &pktlen)
		if err != nil {
			return fmt.Errorf("mux: Error reading pktlen: %w",
				err)
		}
		err = binary.Read(buf, binary.LittleEndian, &ctrl)
		if err != nil {
			return fmt.Errorf("mux: Error reading ctrl: %w", err)
		}
		err = binary.Read(buf, binary.LittleEndian, &stream)
		if err != nil {
			return fmt.Errorf("mux: Error reading stream: %w",
				err)
		}
		databuf = make([]byte, pktlen)

		datalen := uint16(0)
		for datalen < pktlen {
			nn, err := m.c.Read(databuf[datalen:])
			if err != nil {
				return fmt.Errorf("mux: Error reading data: %w",
					err)
			}
			datalen += uint16(nn)

		}
		m.recvCond.L.Lock()
		m.databuf = append(m.databuf, databuf...)
		m.recvCond.L.Unlock()
		m.recvCond.Broadcast()
	}
}

func (m *Mux) Read(p []byte) (n int, err error) {
	m.recvCond.L.Lock()
	for {
		n = len(m.databuf)
		if n != 0 {
			break
		}
		m.recvCond.Wait()
	}
	if n > len(p) {
		n = len(p)
	}
	copy(p, m.databuf)
	m.databuf = m.databuf[n:]
	m.recvCond.L.Unlock()
	return
}

func (m *Mux) Write(p []byte) (n int, err error) {
	pktlen := uint16(len(p))
	ctrl := byte(0)
	stream := byte(0)

	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.LittleEndian, pktlen)
	if err != nil {
		return 0, fmt.Errorf("mux: Error writing pktlen: %w", err)
	}
	err = binary.Write(buf, binary.LittleEndian, ctrl)
	if err != nil {
		return 0, fmt.Errorf("mux: Error writing ctrl: %w", err)
	}
	err = binary.Write(buf, binary.LittleEndian, stream)
	if err != nil {
		return 0, fmt.Errorf("mux: Error writing stream: %w", err)
	}

	o := append(buf.Bytes(), p...)

	for len(o) != 0 {
		nn, err := m.c.Write(o)
		if err != nil {
			return 0, err
		}
		o = o[nn:]
	}
	return len(p), nil
}
