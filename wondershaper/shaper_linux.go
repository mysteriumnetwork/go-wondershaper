package wondershaper

import (
	"io"
	"os/exec"
	"strconv"
	"strings"
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

// LimitDownlink limits download speed of the specified network interface.
func (s Shaper) LimitDownlink(interfaceName string, limitKbps int) error {
	if err := s.installRootHTB(interfaceName); err != nil {
		return err
	}

	// Add the IFB interface
	cmd := s.sudo("modprobe", "ifb", "numifbs=1")
	if err := cmd.Run(); err != nil {
		return err
	}
	cmd = s.sudo("ip", "link", "set", "dev", ifb, "up")
	if err := cmd.Run(); err != nil {
		return err
	}

	// Redirect ingress (incoming) to egress ifb0
	cmd = s.sudo("tc", "qdisc", "add", "dev", interfaceName, "handle", "ffff:", "ingress")
	if err := cmd.Run(); err != nil {
		return err
	}
	cmd = s.sudo("tc", "filter", "add", "dev", interfaceName, "parent", "ffff:", "protocol", "ip", "u32", "match", "u32", "0", "0",
		"action", "mirred", "egress", "redirect", "dev", ifb,
	)
	if err := cmd.Run(); err != nil {
		return err
	}

	// Add class and rules for virtual
	cmd = s.sudo("tc", "qdisc", "add", "dev", ifb, "root", "handle", "2:", "htb")
	if err := cmd.Run(); err != nil {
		return err
	}
	rate := strconv.Itoa(limitKbps) + "kbit"
	cmd = s.sudo("tc", "class", "add", "dev", ifb, "parent", "2:", "classid", "2:1", "htb", "rate", rate)
	if err := cmd.Run(); err != nil {
		return err
	}

	// Add filter to rule for IP address
	cmd = s.sudo("tc", "filter", "add", "dev", ifb, "protocol", "ip", "parent", "2:", "prio", "1", "u32", "match", "ip", "src", "0.0.0.0/0", "flowid", "2:1")
	return cmd.Run()
}

// LimitUplink limits upload speed of the specified network interface.
func (s Shaper) LimitUplink(interfaceName string, limitKbps int) error {
	if err := s.installRootHTB(interfaceName); err != nil {
		return err
	}

	rate := strconv.Itoa(limitKbps) + "kbit"
	cmd := s.sudo("tc", "class", "add", "dev", interfaceName, "parent", "1:", "classid", "1:1", "htb",
		"rate", rate,
		"prio", "5",
	)
	if err := cmd.Run(); err != nil {
		return err
	}

	// High prio class 1:10:
	rate = strconv.Itoa(limitKbps*40/100) + "kbit"
	ceil := strconv.Itoa(limitKbps*95/100) + "kbit"
	cmd = s.sudo("tc", "class", "add", "dev", interfaceName, "parent", "1:1", "classid", "1:10", "htb",
		"rate", rate, "ceil", ceil,
		"prio", "1",
	)
	if err := cmd.Run(); err != nil {
		return err
	}

	// Bulk and default class 1:20 - gets slightly less traffic, and a lower priority
	cmd = s.sudo("tc", "class", "add", "dev", interfaceName, "parent", "1:1", "classid", "1:20", "htb",
		"rate", rate, "ceil", ceil,
		"prio", "2",
	)
	if err := cmd.Run(); err != nil {
		return err
	}

	// 'Traffic we hate'
	rate = strconv.Itoa(limitKbps*20/100) + "kbit"
	ceil = strconv.Itoa(limitKbps*90/100) + "kbit"
	cmd = s.sudo("tc", "class", "add", "dev", interfaceName, "parent", "1:1", "classid", "1:30", "htb", "rate", rate, "ceil", ceil, "prio", "3")
	if err := cmd.Run(); err != nil {
		return err
	}

	// All get Stochastic Fairness
	cmd = s.sudo("tc", "qdisc", "add", "dev", interfaceName, "parent", "1:10", "handle", "10:", "sfq", "perturb", "10", "quantum", quantum)
	if err := cmd.Run(); err != nil {
		return err
	}
	cmd = s.sudo("tc", "qdisc", "add", "dev", interfaceName, "parent", "1:20", "handle", "20:", "sfq", "perturb", "10", "quantum", quantum)
	if err := cmd.Run(); err != nil {
		return err
	}
	cmd = s.sudo("tc", "qdisc", "add", "dev", interfaceName, "parent", "1:30", "handle", "30:", "sfq", "perturb", "10", "quantum", quantum)
	if err := cmd.Run(); err != nil {
		return err
	}

	// Start filters
	// TOS Minimum Delay (ssh, NOT scp) in 1:10:
	cmd = s.sudo("tc", "filter", "add", "dev", interfaceName, "parent", "1:", "protocol", "ip", "prio", "10", "u32",
		"match", "ip", "tos", "0x10", "0xff",
		"flowid", "1:10")
	if err := cmd.Run(); err != nil {
		return err
	}

	// ICMP (ip protocol 1) in the interactive class 1:10 so we can do measurements & impress our friends:
	cmd = s.sudo("tc", "filter", "add", "dev", interfaceName, "parent", "1:", "protocol", "ip", "prio", "11", "u32",
		"match", "ip", "protocol", "1", "0xff",
		"flowid", "1:10")
	if err := cmd.Run(); err != nil {
		return err
	}

	// Prioritize small packets (<64 bytes)
	cmd = s.sudo("tc", "filter", "add", "dev", interfaceName, "parent", "1:", "protocol", "ip", "prio", "12", "u32",
		"match", "ip", "protocol", "6", "0xff",
		"match", "u8", "0x05", "0x0f", "at", "0",
		"match", "u16", "0x0000", "0xffc0", "at", "2",
		"flowid", "1:10")
	if err := cmd.Run(); err != nil {
		return err
	}

	// Some traffic however suffers a worse fate
	cmd = s.sudo("tc", "filter", "add", "dev", interfaceName, "parent", "1:", "protocol", "ip", "prio", "16", "u32",
		"match", "ip", "src", nopriohostsrc,
		"flowid", "1:30")
	if err := cmd.Run(); err != nil {
		return err
	}

	// Rest is 'non-interactive' ie 'bulk' and ends up in 1:20
	cmd = s.sudo("tc", "filter", "add", "dev", interfaceName, "parent", "1:", "protocol", "ip", "prio", "18", "u32",
		"match", "ip", "dst", "0.0.0.0/0",
		"flowid", "1:20")
	return cmd.Run()
}

// Clear clears the limits from the adapter
func (s Shaper) Clear(interfaceName string) {
	_ = s.sudo("tc", "qdisc", "del", "dev", interfaceName, "root").Run()
	_ = s.sudo("tc", "qdisc", "del", "dev", interfaceName, "ingress").Run()
	_ = s.sudo("tc", "qdisc", "del", "dev", ifb, "root").Run()
	_ = s.sudo("tc", "qdisc", "del", "dev", ifb, "ingress").Run()
}

// Status shows the current status of the adapter
func (s Shaper) Status(interfaceName string) error {
	cmd := s.sudo("tc", "-s", "qdisc", "ls", "dev", interfaceName)
	if err := cmd.Run(); err != nil {
		return err
	}
	cmd = s.sudo("tc", "-s", "class", "ls", "dev", interfaceName)
	return cmd.Run()
}

func (s Shaper) installRootHTB(interfaceName string) error {
	out, err := exec.Command("tc", "-s", "qdisc", "ls", "dev", interfaceName).CombinedOutput()
	if err != nil {
		return err
	}

	if strings.Contains(string(out), "qdisc htb 1: root") {
		return nil
	}

	cmd := s.sudo("tc", "qdisc", "add", "dev", interfaceName, "root", "handle", "1:", "htb", "default", "20")
	return cmd.Run()
}

func (s Shaper) sudo(args ...string) *exec.Cmd {
	c := exec.Command("sudo", args...)
	c.Stdout = s.Stdout
	c.Stderr = s.Stderr
	return c
}
