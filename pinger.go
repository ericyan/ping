package ping

import (
	"log"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type Pinger struct {
	id   int
	conn *icmp.PacketConn
}

func NewPinger() (*Pinger, error) {
	id := os.Getpid() & 0xffff
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return nil, err
	}

	return &Pinger{id, conn}, nil
}

func (p *Pinger) Ping(target string) (float64, error) {
	dst := &net.IPAddr{IP: net.ParseIP(target)}

	ts, err := time.Now().MarshalBinary()
	if err != nil {
		return 0, err
	}

	req, err := (&icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   p.id,
			Seq:  1,
			Data: ts,
		},
	}).Marshal(nil)
	if err != nil {
		return 0, err
	}

	if _, err := p.conn.WriteTo(req, dst); err != nil {
		return 0, err
	}

	buf := make([]byte, 1500)
	for {
		n, peer, err := p.conn.ReadFrom(buf)
		if err != nil {
			return 0, err
		}

		// Record receive time asap
		now := time.Now()

		msg, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), buf[:n])
		if err != nil {
			return 0, err
		}

		if msg.Type == ipv4.ICMPTypeEchoReply {
			// The first 32 bits of the ICMP message is not included in icmp.MessageBody
			len := 4 + msg.Body.Len(ipv4.ICMPTypeEchoReply.Protocol())
			reply := msg.Body.(*icmp.Echo)
			log.Printf("%d bytes from %s: icmp_id=%d icmp_seq=%d\n", len, peer, reply.ID, reply.Seq)

			if peer.String() == dst.String() {
				ts := new(time.Time)
				err = ts.UnmarshalBinary(reply.Data)
				if err != nil {
					return 0, err
				}

				rtt := float64(now.Sub(*ts)) / float64(time.Millisecond)
				return rtt, nil
			}
		} else {
			log.Printf("got unknown ICMP message from %s: type=%d\n", peer, msg.Type)
		}
	}
}

func (p *Pinger) Close() error {
	return p.conn.Close()
}
