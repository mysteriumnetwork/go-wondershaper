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
