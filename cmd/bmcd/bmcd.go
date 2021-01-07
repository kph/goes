// Copyright © 2020 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package bmcd

import (
	"fmt"
	"net/rpc"
	"os"
	"os/signal"
	"time"

	"github.com/platinasystems/goes/external/serial"
	"github.com/platinasystems/goes/lang"
)

type Command struct {
}

func (Command) String() string { return "bmcd" }

func (Command) Usage() string {
	return "daemons start bmcd [OPTION]..."
}

func (Command) Apropos() lang.Alt {
	return lang.Alt{
		lang.EnUS: "BMC Daemon",
	}
}

func (Command) Man() lang.Alt {
	return lang.Alt{
		lang.EnUS: `
DESCRIPTION
	Management server on BMC`,
	}
}

func (Command) Main(args ...string) (err error) {
	var dev *os.File

	signal.Reset(os.Interrupt)
	for {
		dev, err = os.OpenFile("/dev/i2c-slave-stream-0",
			os.O_RDWR, 0)
		if err == nil {
			break
		}
		fmt.Printf("Error opening I2C device: %s\n", err)
		time.Sleep(time.Second)
	}
	defer dev.Close()

	transport := serial.NewSequencer(serial.NewCRC(serial.NewFramer(dev)))

	client := rpc.NewClient(transport)

	reply := ""
	for {
		err = client.Call("Swapper.SwapString",
			"The quick brown fox did what?",
			&reply)
		if err != nil {
			fmt.Printf("Error %s\n", err)
		} else {
			fmt.Println(reply)
		}
	}

	return
}
