// Copyright © 2019 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package goes

import (
	"fmt"
	"strconv"
	"syscall"
)

func (g *Goes) findPg(args ...string) (*ProcessGroup, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("not yet")
	}
	jnum, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return nil, err
	}
	if jnum == 0 {
		return nil, fmt.Errorf("not yet")
	}
	if jnum >= uint64(len(g.Jobs)+1) || (g.Jobs[jnum-1] == nil) {
		return nil, fmt.Errorf("No such jnum")
	}
	return g.Jobs[jnum-1], nil

}

func (g *Goes) bg(args ...string) (err error) {
	pg, err := g.findPg(args[0])
	if err != nil {
		return err
	}
	err = syscall.Kill(-pg.Pgid, syscall.SIGCONT)
	if err != nil {
		return err
	}
	for _, pe := range pg.Pe {
		if pe.Ws.Stopped() {
			pe.Ws = 0 // Reset wait state
		}
	}
	return nil
}

func (g *Goes) fg(args ...string) (err error) {
	pg, err := g.findPg(args[0])
	if err != nil {
		return err
	}
	state, err := g.RunInForeground(pg)
	fmt.Printf("Returned state %s error %s\n", state, err)
	return err
}
