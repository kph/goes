// Copyright © 2015-2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package goes

import (
	"fmt"
	"strconv"
)

func (g *Goes) fg(args ...string) (err error) {
	if len(args) != 1 {
		return fmt.Errorf("not yet")
	}
	job, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return err
	}
	if job == 0 {
		return fmt.Errorf("not yet")
	}
	if job >= uint64(len(g.Jobs)+1) || (g.Jobs[job-1] == nil) {
		return fmt.Errorf("No such job")
	}
	pg := g.Jobs[job-1]
	state, err := g.RunInForeground(pg)
	fmt.Printf("Returned state %s error %s\n", state, err)
	return err
}
