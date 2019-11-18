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
 */

package main

import (
	"flag"
	"log"
	"os"

	"github.com/mysteriumnetwork/go-wondershaper/wondershaper"
)

func main() {
	adapter := flag.String("a", "", "set the adapter")
	down := flag.Int("d", 0, "set maximum download rate (in Kbps)")
	up := flag.Int("u", 0, "set maximum upload rate (in Kbps)")
	clear := flag.Bool("c", false, "clear the limits from adapter")
	status := flag.Bool("s", false, "show the current status of adapter")
	flag.Parse()

	if *adapter == "" {
		log.Fatalln("Please supply the adapter name")
	}

	shaper := wondershaper.New()
	shaper.Stdout = os.Stdout
	shaper.Stderr = os.Stderr

	if *clear {
		shaper.Clear(*adapter)
	}

	if *down != 0 {
		err := shaper.LimitDownlink(*adapter, *down)
		if err != nil {
			log.Fatalln("Could not limit downlink", err)
		}
	}

	if *up != 0 {
		err := shaper.LimitUplink(*adapter, *up)
		if err != nil {
			log.Fatalln("Could not limit uplink", err)
		}
	}

	if *status {
		err := shaper.Status(*adapter)
		if err != nil {
			log.Fatalln("Could not query adapter status", err)
		}
	}
}
