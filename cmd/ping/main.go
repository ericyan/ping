package main

import (
	"log"
	"net"
	"os"
	"time"

	"github.com/ericyan/ping"
)

func main() {
	pinger, err := ping.NewICMP()
	if err != nil {
		log.Fatalln(err)
	}
	defer pinger.Close()

	if len(os.Args) < 2 {
		log.Fatalln("please specify at least one target IP")
	}

	for i := 1; i < len(os.Args); i++ {
		target := &net.IPAddr{IP: net.ParseIP(os.Args[i])}
		rtt, err := pinger.Ping(target)
		if err != nil {
			log.Printf("target=%s status=fail reason=%s", target, err)
		} else {
			log.Printf("target=%s status=success rtt=%f", target, float64(rtt)/float64(time.Millisecond))
		}
	}
}
