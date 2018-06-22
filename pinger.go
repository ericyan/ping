package ping

import (
	"errors"
	"log"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ericyan/ping/internal/timestamp"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type message struct {
	t    time.Time
	id   int
	seq  int
	body icmp.MessageBody
	err  error
}

func parseMessage(buf []byte) *message {
	// Record receive time asap
	now := time.Now()

	msg, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), buf)
	if err != nil {
		return &message{now, 0, 0, nil, err}
	}

	switch msg.Type {
	case ipv4.ICMPTypeEchoReply:
		reply, ok := msg.Body.(*icmp.Echo)
		if !ok {
			return &message{now, 0, 0, nil, errors.New("type assertion failed")}
		}

		return &message{now, reply.ID, reply.Seq, msg.Body, nil}
	case ipv4.ICMPTypeEcho:
		// Ignore echo requests
		return &message{now, 0, 0, nil, nil}
	case ipv4.ICMPTypeDestinationUnreachable:
		reply, ok := msg.Body.(*icmp.DstUnreach)
		if !ok {
			return &message{now, 0, 0, nil, errors.New("type assertion failed")}
		}

		msg, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), reply.Data[ipv4.HeaderLen:])
		if err != nil {
			return &message{now, 0, 0, nil, err}
		}
		req := msg.Body.(*icmp.Echo)

		return &message{now, req.ID, req.Seq, msg.Body, errors.New("destination unreachable")}
	case ipv4.ICMPTypeTimeExceeded:
		reply, ok := msg.Body.(*icmp.TimeExceeded)
		if !ok {
			return &message{now, 0, 0, nil, errors.New("type assertion failed")}
		}

		msg, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), reply.Data[ipv4.HeaderLen:])
		if err != nil {
			return &message{now, 0, 0, nil, err}
		}
		req := msg.Body.(*icmp.Echo)

		return &message{now, req.ID, req.Seq, msg.Body, errors.New("time exceeded")}
	default:
		return &message{now, 0, 0, nil, nil}
	}
}

type Pinger struct {
	id   int
	seq  uint64
	conn *icmp.PacketConn
	mu   *sync.Mutex
	recv map[int]chan *message
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
		seq:     0,
		conn:    conn,
		mu:      new(sync.Mutex),
		recv:    make(map[int]chan *message),
		stop:    make(chan bool),
		Timeout: 5000,
	}

	go func(p *Pinger) {
		buf := make([]byte, 1500)
		for {
			select {
			default:
				p.conn.SetReadDeadline(time.Now().Add(time.Duration(p.Timeout) * time.Millisecond))

				n, _, err := p.conn.ReadFrom(buf)
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

				result := parseMessage(buf[:n])
				if result.body != nil || result.err != nil {
					// Ignore messages intended for other pingers
					if result.id != p.id {
						continue
					}

					if c, ok := p.recv[result.seq]; ok {
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
	seq := int(atomic.AddUint64(&p.seq, 1) & 0xffff)

	p.mu.Lock()
	p.recv[seq] = make(chan *message)
	defer func() {
		close(p.recv[seq])
		delete(p.recv, seq)
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
			Seq:  seq,
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
	case reply := <-p.recv[seq]:
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
