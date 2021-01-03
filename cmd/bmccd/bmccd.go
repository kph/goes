// Copyright © 2020 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package bmccd

import (
	"fmt"
	"net/rpc"
	"os"
	"time"

	"github.com/platinasystems/goes/external/serial"
	"github.com/platinasystems/goes/lang"
)

type Command struct {
}

func (Command) String() string { return "bmccd" }

func (Command) Usage() string {
	return "daemons start bmccd [OPTION]..."
}

func (Command) Apropos() lang.Alt {
	return lang.Alt{
		lang.EnUS: "BMC Client Daemon",
	}
}

func (Command) Man() lang.Alt {
	return lang.Alt{
		lang.EnUS: `
DESCRIPTION
	Communicate with management server on BMC`,
	}
}

func (Command) Main(args ...string) (err error) {
	var dev *os.File

	for {
		dev, err = os.OpenFile("/dev/i2c-master-stream-0",
			os.O_RDWR, 0)
		if err == nil {
			break
		}
		fmt.Printf("Error opening I2C device: %s\n", err)
		time.Sleep(time.Second)
	}
	defer dev.Close()
	transport := serial.NewSequencer(serial.NewCRC(serial.NewFramer(dev)))
	srv := rpc.NewServer()
	srv.Register(new(Swapper))

	go func() {
		srv.ServeConn(transport)
		fmt.Println("ServeConn exited")
	}()

	select {}
	return
}

type Swapper struct{}

func (*Swapper) SwapString(in string, out *string) (err error) {
	result := ""

	for _, v := range in {
		result = string(v) + result
	}
	*out = result
	return
}
