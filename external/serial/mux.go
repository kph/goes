// Copyright © 2020 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package serial

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type Mux struct {
	c       io.ReadWriter
	databuf []byte
}

func NewMux(c io.ReadWriter) (s *Mux) {
	s = &Mux{c: c}
	return s
}

func (s *Mux) Read(p []byte) (n int, err error) {
	var pktlen uint16
	var ctrl byte
	var stream byte

	for len(s.databuf) == 0 {
		hdr := make([]byte, 4)
		for hdrlen := 0; hdrlen < 4; {
			nn, err := s.c.Read(hdr[hdrlen:])
			if err != nil {
				return 0, fmt.Errorf("mux: Error reading header: %w",
					err)
			}
			hdrlen += nn
		}

		buf := bytes.NewReader(hdr)
		err = binary.Read(buf, binary.LittleEndian, &pktlen)
		if err != nil {
			return 0, fmt.Errorf("mux: Error reading pktlen: %w",
				err)
		}
		err = binary.Read(buf, binary.LittleEndian, &ctrl)
		if err != nil {
			return 0, fmt.Errorf("mux: Error reading ctrl: %w", err)
		}
		err = binary.Read(buf, binary.LittleEndian, &stream)
		if err != nil {
			return 0, fmt.Errorf("mux: Error reading stream: %w",
				err)
		}
		s.databuf = make([]byte, pktlen)

		datalen := uint16(0)
		for datalen < pktlen {
			nn, err := s.c.Read(s.databuf[datalen:])
			if err != nil {
				return 0, fmt.Errorf("mux: Error reading data: %w",
					err)
			}
			datalen += uint16(nn)
		}
	}

	cnt := int(pktlen)
	if cnt > len(p) {
		cnt = len(p)
	}

	copy(p, s.databuf[:cnt])
	s.databuf = s.databuf[cnt:]

	return cnt, nil
}

func (s *Mux) Write(p []byte) (n int, err error) {
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
		nn, err := s.c.Write(o)
		if err != nil {
			return 0, err
		}
		o = o[nn:]
	}
	return len(p), nil
}
