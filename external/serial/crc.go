// Copyright © 2020 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package serial

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
)

type CRC struct {
	c io.ReadWriter
}

func NewCRC(c io.ReadWriter) (s *CRC) {
	s = &CRC{c: c}
	return s
}

func (s *CRC) Read(p []byte) (n int, err error) {
	readbuf := make([]byte, 1024, 1024)

	nn, err := s.c.Read(readbuf)
	if err != nil || nn < 4 {
		if err == nil {
			err = fmt.Errorf("Bad read length %d", nn)
		}
		return 0, err
	}
	crcBytes := readbuf[nn-4:]
	readbuf = readbuf[:nn-4]

	var checksum uint32
	buf := bytes.NewReader(crcBytes)
	err = binary.Read(buf, binary.LittleEndian, &checksum)
	if err != nil {
		return 0, err
	}
	calc := crc32.ChecksumIEEE(readbuf)
	if calc != checksum {
		return 0, fmt.Errorf("Checksum error calculated %x got %x",
			calc, checksum)
	}
	copy(p, readbuf)
	return len(readbuf), nil
}

func (s *CRC) Write(p []byte) (n int, err error) {
	checksum := crc32.ChecksumIEEE(p)
	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.LittleEndian, checksum)
	if err != nil {
		return 0, fmt.Errorf("Error in binary.Write: %w", err)
	}

	o := append(p, buf.Bytes()...)

	for len(o) != 0 {
		nn, err := s.c.Write(o)
		if err != nil {
			return 0, err
		}
		n = n + nn
		o = o[nn:]
	}
	return
}
