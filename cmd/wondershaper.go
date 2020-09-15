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
