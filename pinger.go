package ping

import (
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/ericyan/ping/internal/timestamp"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type message struct {
	t    time.Time
	body icmp.MessageBody
	err  error
}

type Pinger struct {
	id   int
	conn *icmp.PacketConn
	recv map[string]chan *message
	stop chan bool

	Timeout uint // Timeout in milliseconds
}

func NewPinger() (*Pinger, error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return nil, err
	}
	p := &Pinger{
		id:      int(r.Int63() & 0xffff),
		conn:    conn,
		recv:    make(map[string]chan *message),
		stop:    make(chan bool),
		Timeout: 1000,
	}

	go func(p *Pinger) {
		buf := make([]byte, 1500)
		for {
			select {
			default:
				p.conn.SetReadDeadline(time.Now().Add(time.Duration(p.Timeout) * time.Millisecond))

				n, peer, err := p.conn.ReadFrom(buf)
				if err != nil {
					// Ignore read timeout errors
					if neterr, ok := err.(*net.OpError); ok {
						if neterr.Timeout() {
							continue
						}
					}

					log.Println(err)
					continue
				}

				// Record receive time asap
				now := time.Now()

				msg, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), buf[:n])
				if err != nil {
					log.Println(err)
					continue
				}

				if msg.Type == ipv4.ICMPTypeEchoReply {
					reply := msg.Body.(*icmp.Echo)

					// Ignore messages for other pingers
					if reply.ID != p.id {
						continue
					}

					// The first 32 bits of the ICMP message is not included in icmp.MessageBody
					len := 4 + msg.Body.Len(ipv4.ICMPTypeEchoReply.Protocol())
					log.Printf("%d bytes from %s: icmp_id=%d icmp_seq=%d\n", len, peer, reply.ID, reply.Seq)

					for dst, c := range p.recv {
						if peer.String() == dst {
							c <- &message{now, msg.Body, nil}
						}
					}
				} else {
					log.Printf("got unknown ICMP message from %s: type=%d\n", peer, msg.Type)
				}
			case <-p.stop:
				close(p.stop)
				return
			}
		}
	}(p)

	return p, nil
}

func (p *Pinger) Ping(dst net.Addr) (float64, error) {
	p.recv[dst.String()] = make(chan *message)
	defer func() {
		close(p.recv[dst.String()])
		delete(p.recv, dst.String())
	}()

	payload := make([]byte, 56)
	ts, _ := timestamp.Now().MarshalBinary()
	copy(payload, ts)

	req, err := (&icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   p.id,
			Seq:  1,
			Data: payload,
		},
	}).Marshal(nil)
	if err != nil {
		return 0, err
	}

	if _, err := p.conn.WriteTo(req, dst); err != nil {
		return 0, err
	}

	reply := <-p.recv[dst.String()]
	if reply.err != nil {
		return 0, reply.err
	}
	t := new(timestamp.Timestamp)
	t.UnmarshalBinary(reply.body.(*icmp.Echo).Data[:8])

	rtt := float64(reply.t.Sub(t.Time())) / float64(time.Millisecond)
	return rtt, nil
}

func (p *Pinger) Close() error {
	p.stop <- true
	return p.conn.Close()
}
