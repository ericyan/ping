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
		log.Fatalln("please specify the target IP")
	}
	rtt, err := pinger.Ping(os.Args[1])
	if err != nil {
		log.Fatalf("status=fail reason=%s", err)
	}
	log.Printf("status=success rtt=%f", rtt)
}
