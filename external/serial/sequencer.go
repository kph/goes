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

type Sequencer struct {
	c       io.ReadWriter
	seqXmt  uint16 // Sequence number to transmit
	seqRxmt uint16 // Sequence number retransmit buffer represents
	seqRcv  uint16 // Sequence number we've received
	rxmtBuf []byte // Retransmit buffer
}

func NewSequencer(c io.ReadWriter) (s *Sequencer) {
	s = &Sequencer{c: c}

	return s
}

func (s *Sequencer) Read(p []byte) (n int, err error) {
	for {
		readbuf := make([]byte, 1024, 1024)
		nn, err := s.c.Read(readbuf)
		if err != nil || nn < 4 {
			if err == nil {
				err = fmt.Errorf("Bad read length %d", nn)
			}
			return 0, err
		}
		hdrBytes := readbuf[:4]
		readbuf = readbuf[4:nn]

		var seq uint16
		var ack uint16
		buf := bytes.NewReader(hdrBytes)
		err = binary.Read(buf, binary.LittleEndian, &seq)
		if err != nil {
			return 0, err
		}
		err = binary.Read(buf, binary.LittleEndian, &ack)
		if err != nil {
			return 0, err
		}

		distanceRxmt := int16(ack - s.seqRxmt)
		fmt.Printf("distanceRxmt is %d s.seqRxmt %d\n", distanceRxmt,
			s.seqRxmt)
		if distanceRxmt >= 0 {
			if distanceRxmt > int16(len(s.rxmtBuf)) {
				return 0, fmt.Errorf("distanceRxmt is %d but rxmt is only %d",
					distanceRxmt, len(s.rxmtBuf))
			}
			s.rxmtBuf = s.rxmtBuf[distanceRxmt:]
			s.seqRxmt += uint16(distanceRxmt)
		}

		distanceSeq := int16(seq - s.seqRcv)
		fmt.Printf("distanceSeq is %d s.seqRcv is %d\n", distanceSeq,
			s.seqRcv)
		if distanceSeq < 0 {
			distanceSeq = -distanceSeq
			if distanceSeq <= int16(len(readbuf)) {
				readbuf = readbuf[distanceSeq:]
			}
			distanceSeq = 0
		}
		if distanceSeq > 0 {
			continue // We could cache out of order
		}
		readbuf = readbuf[distanceSeq:]
		n = len(readbuf)
		if n != 0 {
			s.seqRcv += uint16(n)
			copy(p, readbuf)
			return n, nil
		}
	}
}

func (s *Sequencer) Write(p []byte) (n int, err error) {
	s.rxmtBuf = append(s.rxmtBuf, p...)
	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.LittleEndian, s.seqXmt)
	if err != nil {
		return 0, fmt.Errorf("Error in binary.Write: %w", err)
	}

	err = binary.Write(buf, binary.LittleEndian, s.seqRcv)
	if err != nil {
		return 0, fmt.Errorf("Error in binary.Write: %w", err)
	}

	o := append(buf.Bytes(), p...)

	for len(o) != 0 {
		nn, err := s.c.Write(o)
		if err != nil {
			return 0, err
		}
		n = n + nn
		o = o[nn:]
	}

	s.seqXmt += uint16(len(p))
	return
}
