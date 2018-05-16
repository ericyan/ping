package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/ericyan/ping"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	bind     = flag.String("bind", "0.0.0.0", "interface to bind")
	port     = flag.Int("port", 9344, "port to listen on for HTTP requests")
	interval = flag.Int("interval", 3, "seconds to wait between sending each packet")
)

var (
	rttHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ping_rtt_seconds",
			Help:    "Ping round-trip time in seconds.",
			Buckets: []float64{0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.3, 0.5, 1},
		},
		[]string{"target"},
	)
)

func init() {
	prometheus.MustRegister(rttHistogram)
}

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		log.Fatalln("please specify at least one target IP")
	}

	pinger, err := ping.NewPinger()
	if err != nil {
		log.Fatalln(err)
	}
	defer pinger.Close()

	for _, target := range flag.Args() {
		ip := net.ParseIP(target)
		if ip == nil {
			log.Fatalln("invlid target IP:", target)
		}

		go func(ip net.IP) {
			for range time.Tick(time.Duration(*interval) * time.Second) {
				rtt, err := pinger.Ping(&net.IPAddr{IP: ip})
				if err != nil {
					log.Printf("target=%s status=fail reason=%s", ip.String(), err)
				} else {
					rttHistogram.With(prometheus.Labels{"target": ip.String()}).Observe(rtt / 1000)
				}
			}
		}(ip)
	}

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(*bind+":"+strconv.Itoa(*port), nil))
}