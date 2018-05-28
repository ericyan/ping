package ping

import (
	"errors"
	"log"
	"math/rand"
	"net"
	"sync"
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

func parseMessage(id int, buf []byte) *message {
	// Record receive time asap
	now := time.Now()

	msg, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), buf)
	if err != nil {
		return &message{now, nil, err}
	}

	switch msg.Type {
	case ipv4.ICMPTypeEchoReply:
		reply := msg.Body.(*icmp.Echo)

		// Ignore messages for other pingers
		if reply.ID != id {
			return &message{now, nil, nil}
		}

		return &message{now, msg.Body, nil}
	case ipv4.ICMPTypeEcho:
		// Ignore echo requests
		return &message{now, nil, nil}
	case ipv4.ICMPTypeDestinationUnreachable:
		reply := msg.Body.(*icmp.DstUnreach)
		msg, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), reply.Data[ipv4.HeaderLen:])
		if err != nil {
			return &message{now, nil, err}
		}
		req := msg.Body.(*icmp.Echo)

		// Ignore messages for other pingers
		if req.ID != id {
			return &message{now, nil, nil}
		}

		switch msg.Code {
		case 0:
			err = errors.New("net unreachable")
		case 1:
			err = errors.New("host unreachable")
		case 2:
			err = errors.New("protocol unreachable")
		case 3:
			err = errors.New("port unreachable")
		case 4:
			err = errors.New("fragmentation needed")
		case 5:
			err = errors.New("source route failed")
		default:
			err = errors.New("destination unreachable")
		}
		return &message{now, msg.Body, err}
	case ipv4.ICMPTypeTimeExceeded:
		reply := msg.Body.(*icmp.TimeExceeded)
		msg, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), reply.Data[ipv4.HeaderLen:])
		if err != nil {
			return &message{now, nil, err}
		}
		req := msg.Body.(*icmp.Echo)

		// Ignore messages for other pingers
		if req.ID != id {
			return &message{now, nil, nil}
		}

		switch msg.Code {
		case 0:
			err = errors.New("TTL exceeded in transit")
		case 1:
			err = errors.New("fragment reassembly time exceeded")
		default:
			err = errors.New("time exceeded")
		}
		return &message{now, msg.Body, err}
	default:
		return &message{now, nil, nil}
	}
}

type Pinger struct {
	id   int
	conn *icmp.PacketConn
	mu   *sync.Mutex
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
		mu:      new(sync.Mutex),
		recv:    make(map[string]chan *message),
		stop:    make(chan bool),
		Timeout: 3000,
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

				result := parseMessage(p.id, buf[:n])
				if result.body != nil || result.err != nil {
					if c, ok := p.recv[peer.String()]; ok {
						c <- result
					}
				}
			case <-p.stop:
				close(p.stop)
				return
			}
		}
	}(p)

	return p, nil
}

func (p *Pinger) Ping(dst net.Addr) (time.Duration, error) {
	p.mu.Lock()
	p.recv[dst.String()] = make(chan *message)
	defer func() {
		close(p.recv[dst.String()])
		delete(p.recv, dst.String())
		p.mu.Unlock()
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

	select {
	case reply := <-p.recv[dst.String()]:
		if reply.err != nil {
			return 0, reply.err
		}
		t := new(timestamp.Timestamp)
		t.UnmarshalBinary(reply.body.(*icmp.Echo).Data[:8])

		return reply.t.Sub(t.Time()), nil
	case <-time.After(time.Duration(p.Timeout) * time.Millisecond):
		return 0, errors.New("timeout")
	}
}

func (p *Pinger) Close() error {
	p.stop <- true
	return p.conn.Close()
}
