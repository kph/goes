// Copyright © 2020 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package install

import (
	"fmt"
	"net"
	"testing"
)

func TestInstallCommand(t *testing.T) {
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Error("net.Interfaces() failed: ", err)
		return
	}

	for _, iface := range ifaces {
		fmt.Printf("Interface: %v\n", iface)
		addrs, err := iface.Addrs()
		if err != nil {
			t.Error("iface.Addrs() failed: ", err)
			return
		}
		for _, addr := range addrs {
			fmt.Printf("  Addr: %s %s\n", addr.Network(),
				addr.String())
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				ip := ipnet.IP
				if ipv4 := ip.To4(); ipv4 != nil {
					fmt.Printf("   IP4Addr: %s\n", ipnet)
				} else {
					fmt.Printf("   IP6Addr: %s\n", ipnet)
				}
			}
		}
	}
}

func TestDefaultGateway(t *testing.T) {
	gw, iface, err := defaultGateway()
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("Default gw %s via %s\n", gw, iface)
}

func TestDefaultGatewayV6(t *testing.T) {
	gw, iface, err := defaultGatewayV6()
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("Default gw %s via %s\n", gw, iface)
}
