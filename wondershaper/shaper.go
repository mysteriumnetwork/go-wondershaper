/*
 * Copyright (C) 2019 The "MysteriumNetwork/go-wondershaper" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package wondershaper

import (
	"io"
	"os/exec"
	"strconv"
)

const (
	quantum       = "6000"
	ifb           = "ifb0"
	nopriohostsrc = "80"
)

type Shaper struct {
	Stdout, Stderr io.Writer
}

func New() *Shaper {
	return &Shaper{}
}

type LimitOptions struct {
	downloadKbps uint
	uploadKbps   uint
}

func (s Shaper) Limit(interfaceName string, options LimitOptions) error {
	return nil
}

// LimitUplink limits upload speed of the specified network interface.
func (s Shaper) LimitUplink(interfaceName string, limitKbps int) error {
	// Install root HTB
	cmd := s.cmd("tc", "qdisc", "add", "dev", interfaceName, "root", "handle", "1:", "htb",
		"default", "20")
	if err := cmd.Run(); err != nil {
		return err
	}

	rate := strconv.Itoa(limitKbps) + "kbit"
	cmd = s.cmd("tc", "class", "add", "dev", interfaceName, "parent", "1:", "classid", "1:1", "htb",
		"rate", rate,
		"prio", "5",
	)
	if err := cmd.Run(); err != nil {
		return err
	}

	// High prio class 1:10:
	rate = strconv.Itoa(limitKbps*40/100) + "kbit"
	ceil := strconv.Itoa(limitKbps*95/100) + "kbit"
	cmd = s.cmd("tc", "class", "add", "dev", interfaceName, "parent", "1:1", "classid", "1:10", "htb",
		"rate", rate, "ceil", ceil,
		"prio", "1",
	)
	if err := cmd.Run(); err != nil {
		return err
	}

	// Bulk and default class 1:20 - gets slightly less traffic, and a lower priority
	cmd = s.cmd("tc", "class", "add", "dev", interfaceName, "parent", "1:1", "classid", "1:20", "htb",
		"rate", rate, "ceil", ceil,
		"prio", "2",
	)
	if err := cmd.Run(); err != nil {
		return err
	}

	// 'Traffic we hate'
	rate = strconv.Itoa(limitKbps*20/100) + "kbit"
	ceil = strconv.Itoa(limitKbps*90/100) + "kbit"
	cmd = s.cmd("tc", "class", "add", "dev", interfaceName, "parent", "1:1", "classid", "1:30", "htb", "rate", rate, "ceil", ceil, "prio", "3")
	if err := cmd.Run(); err != nil {
		return err
	}

	// All get Stochastic Fairness
	cmd = s.cmd("tc", "qdisc", "add", "dev", interfaceName, "parent", "1:10", "handle", "10:", "sfq", "perturb", "10", "quantum", quantum)
	if err := cmd.Run(); err != nil {
		return err
	}
	cmd = s.cmd("tc", "qdisc", "add", "dev", interfaceName, "parent", "1:20", "handle", "20:", "sfq", "perturb", "10", "quantum", quantum)
	if err := cmd.Run(); err != nil {
		return err
	}
	cmd = s.cmd("tc", "qdisc", "add", "dev", interfaceName, "parent", "1:30", "handle", "30:", "sfq", "perturb", "10", "quantum", quantum)
	if err := cmd.Run(); err != nil {
		return err
	}

	// Start filters
	// TOS Minimum Delay (ssh, NOT scp) in 1:10:
	cmd = s.cmd("tc", "filter", "add", "dev", interfaceName, "parent", "1:", "protocol", "ip", "prio", "10", "u32",
		"match", "ip", "tos", "0x10", "0xff",
		"flowid", "1:10")
	if err := cmd.Run(); err != nil {
		return err
	}

	// ICMP (ip protocol 1) in the interactive class 1:10 so we can do measurements & impress our friends:
	cmd = s.cmd("tc", "filter", "add", "dev", interfaceName, "parent", "1:", "protocol", "ip", "prio", "11", "u32",
		"match", "ip", "protocol", "1", "0xff",
		"flowid", "1:10")
	if err := cmd.Run(); err != nil {
		return err
	}

	// Prioritize small packets (<64 bytes)
	cmd = s.cmd("tc", "filter", "add", "dev", interfaceName, "parent", "1:", "protocol", "ip", "prio", "12", "u32",
		"match", "ip", "protocol", "6", "0xff",
		"match", "u8", "0x05", "0x0f", "at", "0",
		"match", "u16", "0x0000", "0xffc0", "at", "2",
		"flowid", "1:10")
	if err := cmd.Run(); err != nil {
		return err
	}

	// Some traffic however suffers a worse fate
	cmd = s.cmd("tc", "filter", "add", "dev", interfaceName, "parent", "1:", "protocol", "ip", "prio", "16", "u32",
		"match", "ip", "src", nopriohostsrc,
		"flowid", "1:30")
	if err := cmd.Run(); err != nil {
		return err
	}

	// Rest is 'non-interactive' ie 'bulk' and ends up in 1:20
	cmd = s.cmd("tc", "filter", "add", "dev", interfaceName, "parent", "1:", "protocol", "ip", "prio", "18", "u32",
		"match", "ip", "dst", "0.0.0.0/0",
		"flowid", "1:20")
	return cmd.Run()
}

func (s Shaper) Clear(interfaceName string) error {
	return nil
}

// Status shows the current status of the adapter
func (s Shaper) Status(interfaceName string) error {
	cmd := s.cmd("tc", "-s", "qdisc", "ls", "dev", interfaceName)
	if err := cmd.Run(); err != nil {
		return err
	}
	cmd = s.cmd("tc", "-s", "class", "ls", "dev", interfaceName)
	return cmd.Run()
}

func (s Shaper) cmd(name string, args ...string) *exec.Cmd {
	c := exec.Command(name, args...)
	c.Stdout = s.Stdout
	c.Stderr = s.Stderr
	return c
}
