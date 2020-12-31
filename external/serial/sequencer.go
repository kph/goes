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
	"time"
)

type HeaderFlags uint16

const (
	HFInit1   HeaderFlags = 1 << iota // Initialization Phase 1
	HFInit2                           // Initialization Phase 2
	HFRunning                         // Running connection
	HFClose1                          // Close phase 1
	HFClose2                          // Close phase 2
)

type Sequencer struct {
	c        io.ReadWriter
	closed   chan struct{} // Channel closed notifier
	xmitCond *sync.Cond
	recvCond *sync.Cond
	flags    HeaderFlags // Current header flags (state)
	seqXmt   uint16      // Sequence number to transmit
	seqRxmt  uint16      // Sequence number retransmit buffer represents
	seqRcv   uint16      // Sequence number we've received
	lastRcv  time.Time   // Time last packet was received
	rxmtBuf  []byte      // Retransmit buffer
	recvBuf  []byte      // Receive buffer
}

func NewSequencer(c io.ReadWriter) (s *Sequencer) {
	s = &Sequencer{
		c:        c,
		closed:   make(chan struct{}),
		xmitCond: sync.NewCond(&sync.Mutex{}),
		recvCond: sync.NewCond(&sync.Mutex{}),
		flags:    HFInit1,
	}
	go s.runTimer()

	return s
}

func (s *Sequencer) runTimer() {
	go s.backgroundRead()
	for {
		time.Sleep(time.Second)

		select {
		case <-s.closed:
			return
		default:
		}

		//if len(s.rxmtBuf) == 0 {
		//	continue
		//}
		s.recvCond.L.Lock()
		ack := s.seqRcv
		s.recvCond.L.Unlock()

		s.xmitCond.L.Lock()
		fmt.Printf("Sequencer.runTimer: len(s.rxmtBuf)=%d seqXmt=%d seqRxmt=%d seqRcv=%d\n",
			len(s.rxmtBuf), s.seqXmt, s.seqRxmt, ack)

		seq := s.seqRxmt

		buf := new(bytes.Buffer)

		err := binary.Write(buf, binary.LittleEndian, s.flags)
		if err != nil {
			return
		}

		err = binary.Write(buf, binary.LittleEndian, seq)
		if err != nil {
			return
		}

		err = binary.Write(buf, binary.LittleEndian, ack)
		if err != nil {
			return
		}

		err = binary.Write(buf, binary.LittleEndian, int16(len(s.rxmtBuf)))
		if err != nil {
			return
		}

		o := append(buf.Bytes(), s.rxmtBuf...)

		s.xmitCond.L.Unlock()

		for len(o) != 0 {
			nn, err := s.c.Write(o)
			if err != nil {
				return
			}
			o = o[nn:]
		}

		fmt.Printf("Sent retransmit of seq %d ack %d s.seqXmt %d s.seqRxmt %d s.seqRcv %d len(s.rxmtBuf) %d\n",
			seq, ack, s.seqXmt, s.seqRxmt, s.seqRcv, len(s.rxmtBuf))
	}
}

func (s *Sequencer) backgroundRead() {
	readbuf := make([]byte, 1024, 1024)

	for {
		select {
		case <-s.closed:
			return
		default:
		}
		nn, err := s.c.Read(readbuf)
		fmt.Printf("s.c.Read(readbuf()) returned len %d err %s\n", nn,
			err)
		if err != nil || nn < 8 {
			if err == nil {
				err = fmt.Errorf("Bad read length %d", nn)
			}
			fmt.Printf("Exiting backgroundRead: error reading readbuf: %s\n",
				err)
			return
		}
		hdrBytes := readbuf[:8]
		dataBuf := readbuf[8:nn]

		var flags uint16
		var seq uint16
		var ack uint16
		var msgLen uint16

		buf := bytes.NewReader(hdrBytes)
		err = binary.Read(buf, binary.LittleEndian, &flags)
		if err != nil {
			fmt.Printf("Exiting backgroundRead: error reading flags: %s\n",
				err)
			return
		}
		err = binary.Read(buf, binary.LittleEndian, &seq)
		if err != nil {
			fmt.Printf("Exiting backgroundRead: error reading seq: %s\n",
				err)
			return
		}
		err = binary.Read(buf, binary.LittleEndian, &ack)
		if err != nil {
			fmt.Printf("Exiting backgroundRead: error reading ack: %s\n",
				err)
			return
		}
		err = binary.Read(buf, binary.LittleEndian, &msgLen)
		if err != nil {
			fmt.Printf("Exiting backgroundRead: error reading msgLen: %s\n",
				err)
			return
		}
		fmt.Printf("backgroundRead: seq %d ack %d len %d seqRxmt %d seqRcv %d\n",
			seq, ack, msgLen, s.seqRxmt, s.seqRcv)

		s.xmitCond.L.Lock()
		uDistanceRxmt := ack - s.seqRxmt
		distanceRxmt := int16(uDistanceRxmt)
		fmt.Printf("backgroundRead: ack is %d seq is %d uDistanceRxmt is %d distanceRxmt is %d s.seqRxmt %d s.seqXmt %d\n",
			ack, seq, uDistanceRxmt, distanceRxmt, s.seqRxmt, s.seqXmt)
		if distanceRxmt >= 0 {
			if distanceRxmt > int16(len(s.rxmtBuf)) {
				fmt.Printf("Exiting BackgroundRead: distanceRxmt (%d) > len(s.rxmtBuf) (%d)\n",
					distanceRxmt,
					int16(len(s.rxmtBuf)))
				return
			}
			s.rxmtBuf = s.rxmtBuf[distanceRxmt:]
			s.seqRxmt += uint16(distanceRxmt)
			fmt.Printf("Updated received sequence by %d: len(s.rxmtBuf) = %d s.seqRxmt = %d\n",
				distanceRxmt, len(s.rxmtBuf), s.seqRxmt)
		}
		s.xmitCond.L.Unlock()

		s.recvCond.L.Lock()
		distanceSeq := int16(seq - s.seqRcv)
		fmt.Printf("distanceSeq is %d seq is %d s.seqRcv is %d len(dataBuf) is %d\n",
			distanceSeq, seq, s.seqRcv, len(dataBuf))
		if distanceSeq >= 0 {
			if distanceSeq <= int16(len(dataBuf)) {
				fmt.Printf("Databuf before shrink: %s\n",
					dataBuf)
				dataBuf = dataBuf[distanceSeq:]
				fmt.Printf("Databuf after shrink: %s\n",
					dataBuf)
			}
			fmt.Printf("backgroundRead: s.recvBuf %s databuf %s\n",
				s.recvBuf, dataBuf)
			s.lastRcv = time.Now()
			s.recvBuf = append(s.recvBuf, dataBuf...)
			s.seqRcv += uint16(len(dataBuf))
			s.recvCond.Broadcast()
		}

		s.recvCond.L.Unlock()
	}
}

func (s *Sequencer) Read(p []byte) (n int, err error) {
	for {
		s.recvCond.L.Lock()
		for {
			n = len(s.recvBuf)
			fmt.Printf("Sequencer.Read: len(p)=%d len(s.recvBuf) %d\n",
				len(p), n)
			if n != 0 {
				break
			}
			//			if n == 0 {
			//	s.recvCond.L.Unlock()
			//	return 0, io.EOF
			//}
			s.recvCond.Wait()
		}
		if n > len(p) {
			n = len(p)
		}
		fmt.Printf("Sequencer.Read: len(s.recvBuf) %d s.recvBuf %s\n",
			len(s.recvBuf), s.recvBuf)
		copy(p, s.recvBuf)
		s.recvBuf = s.recvBuf[n:]
		s.recvCond.L.Unlock()
		fmt.Printf("Returning Sequencer.Read: len(s.recvBuf) %d\n",
			len(s.recvBuf))
		return n, nil
	}
}

func (s *Sequencer) Write(p []byte) (n int, err error) {
	s.recvCond.L.Lock()
	ack := s.seqRcv
	s.recvCond.L.Unlock()

	s.xmitCond.L.Lock()
	fmt.Printf("Sequencer.Write: len(s.rxmtBuf)=%d len(p)=%d seqXmt=%d seqRxmt=%d seqRcv=%d\n",
		len(s.rxmtBuf), len(p), s.seqXmt, s.seqRxmt, ack)
	seq := s.seqXmt
	s.seqXmt += uint16(len(p))
	s.rxmtBuf = append(s.rxmtBuf, p...)

	if len(s.rxmtBuf) != len(p) {
		s.xmitCond.L.Unlock()
		return len(p), nil
	}

	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.LittleEndian, s.flags)
	if err != nil {
		return
	}
	err = binary.Write(buf, binary.LittleEndian, seq)
	if err != nil {
		return 0, fmt.Errorf("Error in binary.Write: %w", err)
	}

	err = binary.Write(buf, binary.LittleEndian, ack)
	if err != nil {
		return 0, fmt.Errorf("Error in binary.Write: %w", err)
	}

	err = binary.Write(buf, binary.LittleEndian, int16(len(p)))
	if err != nil {
		return 0, fmt.Errorf("Error in binary.Write: %w", err)
	}

	o := append(buf.Bytes(), p...)

	s.xmitCond.L.Unlock()

	for len(o) != 0 {
		nn, err := s.c.Write(o)
		if err != nil {
			return 0, err
		}
		o = o[nn:]
	}

	return len(p), nil
}
