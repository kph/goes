// Copyright © 2015-2020 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

// Package ucd9090 provides access to the UCD9090 Power Sequencer/Monitor chip
package i2cd

import (
	"net"
	"net/http"
	"net/rpc"
	"sync"
	"time"

	"github.com/platinasystems/goes/cmd"
	"github.com/platinasystems/goes/external/i2c"
	"github.com/platinasystems/goes/external/log"
	"github.com/platinasystems/goes/external/redis"
	"github.com/platinasystems/goes/lang"
	"github.com/platinasystems/gpio"
	"github.com/platinasystems/ioport"
)

type Command struct {
	done chan struct{}
}

func (*Command) String() string { return "i2cd" }

func (*Command) Usage() string { return "i2cd" }

func (*Command) Apropos() lang.Alt {
	return lang.Alt{
		lang.EnUS: "i2c server daemon",
	}
}

func (c *Command) Close() error {
	close(c.done)
	return nil
}

func (*Command) Kind() cmd.Kind { return cmd.Daemon }

func (c *Command) Main(...string) error {
	c.done = make(chan struct{})
	i2cReq := &I2cReq{c}
	rpc.Register(i2cReq)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":1233")
	if e != nil {
		log.Print("listen error:", e)
	}
	log.Print("listen OKAY")
	go http.Serve(l, nil)

	for {
		select {
		case <-c.done:
			return nil
		}
	}
	return nil
}

const MAXOPS = 30

type I struct {
	InUse     bool
	RW        i2c.RW
	RegOffset uint8
	BusSize   i2c.SMBusSize
	Data      [i2c.BlockMax]byte
	Bus       int
	Addr      int
	Delay     int
}
type R struct {
	D [i2c.BlockMax]byte
	E error
}

type I2cReq struct {
	c *Command
}

var b = [i2c.BlockMax]byte{0}
var i = I{false, i2c.RW(0), 0, 0, b, 0, 0, 0}
var j [MAXOPS]I
var r = R{b, nil}
var s [MAXOPS]R
var x int
var stopped byte = 0
var mutex = &sync.Mutex{}

func (t *I2cReq) ReadWrite(g *[MAXOPS]I, f *[MAXOPS]R) error {
	mutex.Lock()
	defer mutex.Unlock()

	var bus i2c.Bus
	var data i2c.SMBusData
	if g[0].Bus == 0x99 {
		stopped = byte(g[0].Addr)
		return nil
	}
	if g[0].Bus == 0x98 {
		f[0].D[0] = stopped
		return nil
	}
	for x := 0; x < MAXOPS; x++ {
		if g[x].InUse == true {
			err := bus.Open(g[x].Bus)
			if err != nil {
				log.Print("Error opening I2C bus")
				return err
			}
			defer bus.Close()

			err = bus.ForceSlaveAddress(g[x].Addr)
			if err != nil {
				log.Print("ERR2")
				log.Print("Error setting I2C slave address")
				return err
			}
			data[0] = g[x].Data[0]
			data[1] = g[x].Data[1]
			data[2] = g[x].Data[2]
			data[3] = g[x].Data[3]
			err = bus.Do(g[x].RW, g[x].RegOffset, g[x].BusSize, &data)
			if err != nil {
				for y := 0; y < x; y++ {
					log.Printf("I2C R/W before Error: bus 0x%x addr 0x%x offset 0x%x data 0x%x RW %d BusSize %d delay %d", g[y].Bus, g[y].Addr, g[y].RegOffset, g[y].Data[0], g[y].RW, g[y].BusSize, g[y].Delay)
				}
				log.Printf("Error doing I2C R/W: bus 0x%x addr 0x%x offset 0x%x data 0x%x RW %d BusSize %d delay %d", g[x].Bus, g[x].Addr, g[x].RegOffset, data[0], g[x].RW, g[x].BusSize, g[x].Delay)
				m, _ := redis.Hget(redis.DefaultHash, "machine")

				switch m {
				case "platina-mk1":
					d, err := ioport.Inb(0x603)
					if err == nil {
						ioport.Outb(0x603, d&0xb0)
						time.Sleep(10 * time.Microsecond)
						ioport.Outb(0x603, d|0x40)
					}
				case "platina-mk1-bmc":
					pin, found := gpio.FindPin("FRU_I2C_MUX_RST_L")
					if found {
						pin.SetValue(false)
						time.Sleep(10 * time.Microsecond)
						pin.SetValue(true)
					}

					pin, found = gpio.FindPin("MAIN_I2C_MUX_RST_L")
					if found {
						pin.SetValue(false)
						time.Sleep(10 * time.Microsecond)
						pin.SetValue(true)
					}
				default:
				}
				return err
			}
			f[x].D[0] = data[0]
			f[x].D[1] = data[1]
			if g[x].BusSize == i2c.I2CBlockData {
				for y := 2; y < i2c.BlockMax; y++ {
					f[x].D[y] = data[y]
				}
			}
			bus.Close()
			if g[x].Delay > 0 {
				time.Sleep(time.Duration(g[x].Delay) * time.Millisecond)
			}
		}
	}
	return nil
}

func clearJS() {
	x = 0
	for k := 0; k < MAXOPS; k++ {
		j[k] = i
		s[k] = r
	}
}
