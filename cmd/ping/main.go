package main

import (
	"flag"
	"log"
	"net"
	"time"

	"github.com/ericyan/ping"
)

var (
	icmp = flag.Bool("icmp", true, "use ICMP ping")
	tcp  = flag.Bool("tcp", false, "use TCP ping")
)

func main() {
	flag.Parse()

	var pinger ping.Pinger
	var err error
	if *tcp {
		pinger, err = ping.NewTCP()
	} else {
		pinger, err = ping.NewICMP()
	}
	if err != nil {
		log.Fatalln(err)
	}
	defer pinger.Close()

	if flag.NArg() < 1 {
		log.Fatalln("please specify at least one target IP")
	}

	for i := 0; i < flag.NArg(); i++ {
		var dst net.Addr
		if *tcp {
			dst, err = net.ResolveTCPAddr("tcp4", flag.Arg(i))
		} else {
			dst, err = net.ResolveIPAddr("ip4", flag.Arg(i))
		}
		if err != nil {
			log.Fatalln(err)
		}

		rtt, err := pinger.Ping(dst)
		if err != nil {
			log.Printf("dst=%s status=fail reason=%s", dst, err)
		} else {
			log.Printf("dst=%s status=success rtt=%f", dst, float64(rtt)/float64(time.Millisecond))
		}
	}
}
