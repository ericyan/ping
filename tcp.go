package ping

import (
	"errors"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

type tcpPacket struct {
	t   time.Time
	err error
}

type tx struct {
	t  time.Time
	ch chan *tcpPacket
}

type tcpPinger struct {
	conn    *net.IPConn
	port    uint16
	mu      *sync.Mutex
	recv    map[uint32]*tx
	stop    chan bool
	timeout uint
}

func NewTCP() (Pinger, error) {
	addr, err := net.ResolveIPAddr("ip4:tcp", "127.0.0.1")
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenIP("ip4:tcp", addr)
	if err != nil {
		return nil, err
	}

	p := &tcpPinger{
		conn:    conn,
		port:    23333,
		mu:      new(sync.Mutex),
		recv:    make(map[uint32]*tx),
		stop:    make(chan bool),
		timeout: 5000,
	}

	go func(p *tcpPinger) {
		buf := make([]byte, 1500)
		for {
			select {
			default:
				p.conn.SetReadDeadline(time.Now().Add(time.Duration(p.timeout) * time.Millisecond))

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

				now := time.Now()
				packet := gopacket.NewPacket(buf[:n], layers.LayerTypeTCP, gopacket.Default)
				if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
					tcp := tcpLayer.(*layers.TCP)

					if tcp.DstPort != layers.TCPPort(p.port) {
						continue
					}

					if c, ok := p.recv[tcp.Ack-1]; ok {
						if tcp.SYN && tcp.ACK {
							c.ch <- &tcpPacket{now, nil}
						} else {
							c.ch <- &tcpPacket{now, errors.New("port closed")}
						}
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

func (p *tcpPinger) Ping(dst net.Addr) (time.Duration, error) {
	host, port, err := net.SplitHostPort(dst.String())
	if err != nil {
		return 0, err
	}
	dstAddr, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		return 0, err
	}
	dstPort, err := strconv.Atoi(port)
	if err != nil {
		return 0, err
	}

	seq := uint32(123456789)

	p.mu.Lock()
	p.recv[seq] = &tx{time.Now(), make(chan *tcpPacket)}
	defer func() {
		close(p.recv[seq].ch)
		delete(p.recv, seq)
		p.mu.Unlock()
	}()

	syn := &layers.TCP{
		SrcPort: layers.TCPPort(p.port),
		DstPort: layers.TCPPort(dstPort),
		Seq:     seq,
		SYN:     true,
	}
	syn.SetNetworkLayerForChecksum(&layers.IPv4{
		SrcIP:    p.conn.LocalAddr().(*net.IPAddr).IP,
		DstIP:    dstAddr.IP,
		Protocol: layers.IPProtocolTCP,
	})

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}
	if err := gopacket.SerializeLayers(buf, opts, syn); err != nil {
		return 0, err
	}

	if _, err := p.conn.WriteTo(buf.Bytes(), dstAddr); err != nil {
		return 0, err
	}

	select {
	case reply := <-p.recv[seq].ch:
		rtt := reply.t.Sub(p.recv[seq].t)
		return rtt, reply.err
	case <-time.After(time.Duration(p.timeout) * time.Millisecond):
		return 0, errors.New("timeout")
	}
}

func (p *tcpPinger) Close() error {
	p.stop <- true
	return p.conn.Close()
}
