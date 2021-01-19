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
	streams  [256]Stream
	listener func(stream int) error
}

type Stream struct {
	mux      *Mux
	id       uint8
	recvBuf  []byte
	recvCond *sync.Cond
}

func NewMux(c io.ReadWriter, listener func(stream int) (err error)) (m *Mux) {
	m = &Mux{
		c:        c,
		listener: listener,
	}
	go m.backgroundRead()
	return m
}

func (m *Mux) NewStream(id uint8) (s *Stream) {
	s = &m.streams[id]
	s.mux = m
	s.recvCond = sync.NewCond(&sync.Mutex{})
	s.id = id
	return
}

func (m *Mux) backgroundRead() {
	err := m.doBackgroundRead()
	if err != nil {
		panic(err)
	}
}

func (m *Mux) doBackgroundRead() (err error) {
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
		s := m.streams[stream]
		s.recvCond.L.Lock()
		s.recvBuf = append(s.recvBuf, databuf...)
		s.recvCond.L.Unlock()
		s.recvCond.Broadcast()
	}
}

func (s *Stream) Read(p []byte) (n int, err error) {
	s.recvCond.L.Lock()
	for {
		n = len(s.recvBuf)
		if n != 0 {
			break
		}
		s.recvCond.Wait()
	}
	if n > len(p) {
		n = len(p)
	}
	copy(p, s.recvBuf)
	s.recvBuf = s.recvBuf[n:]
	s.recvCond.L.Unlock()
	return
}

func (s *Stream) Write(p []byte) (n int, err error) {
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
		nn, err := s.mux.c.Write(o)
		if err != nil {
			return 0, err
		}
		o = o[nn:]
	}
	return len(p), nil
}
