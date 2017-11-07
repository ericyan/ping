package main

import (
	"log"
	"os"

	"github.com/ericyan/ping"
)

func main() {
	pinger, err := ping.NewPinger()
	if err != nil {
		log.Fatalln(err)
	}
	defer pinger.Close()

	if len(os.Args) < 2 {
		log.Fatalln("please specify at least one target IP")
	}

	for i := 1; i < len(os.Args); i++ {
		target := os.Args[i]
		rtt, err := pinger.Ping(target)
		if err != nil {
			log.Printf("target=%s status=fail reason=%s", err, target)
		}
		log.Printf("target=%s status=success rtt=%f", target, rtt)
	}
}
