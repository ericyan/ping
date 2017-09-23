package main

import (
	"log"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type Pinger struct {
	conn *icmp.PacketConn
}

func NewPinger() (*Pinger, error) {
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return nil, err
	}

	return &Pinger{conn}, nil
}

func (p *Pinger) Ping(target string) error {
	dst := &net.IPAddr{IP: net.ParseIP(target)}

	ts, err := time.Now().MarshalBinary()
	if err != nil {
		return err
	}

	req, err := (&icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getpid() & 0xffff,
			Seq:  1,
			Data: ts,
		},
	}).Marshal(nil)
	if err != nil {
		return err
	}

	if _, err := p.conn.WriteTo(req, dst); err != nil {
		return err
	}

	buf := make([]byte, 1500)
	for {
		n, peer, err := p.conn.ReadFrom(buf)
		if err != nil {
			log.Println(err)
		}

		// Record receive time asap
		now := time.Now()

		msg, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), buf[:n])
		if err != nil {
			log.Println(err)
		}

		if msg.Type == ipv4.ICMPTypeEchoReply {
			// The first 32 bits of the ICMP message is not included in icmp.MessageBody
			len := 4 + msg.Body.Len(ipv4.ICMPTypeEchoReply.Protocol())
			reply := msg.Body.(*icmp.Echo)

			if peer.String() == dst.String() {
				var rtt float64
				ts := new(time.Time)
				err = ts.UnmarshalBinary(reply.Data)
				if err != nil {
					log.Println(err)
					rtt = -1
				}
				rtt = float64(now.Sub(*ts)) / float64(time.Millisecond)

				log.Printf("%d bytes from %s: icmp_id=%d icmp_seq=%d rtt=%f\n", len, peer, reply.ID, reply.Seq, rtt)
				break
			} else {
				log.Printf("%d bytes from %s (irrelevant): icmp_id=%d icmp_seq=%d\n", len, peer, reply.ID, reply.Seq)
			}
		} else {
			log.Printf("got unknown ICMP message from %s: type=%d\n", peer, msg.Type)
		}
	}

	return nil
}

func (p *Pinger) Close() error {
	return p.conn.Close()
}

func main() {
	pinger, err := NewPinger()
	if err != nil {
		log.Fatalln(err)
	}
	defer pinger.Close()

	if len(os.Args) < 2 {
		log.Fatalln("please specify the target IP")
	}
	err = pinger.Ping(os.Args[1])
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Success")
}
