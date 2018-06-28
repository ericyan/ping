package ping

import (
	"net"
	"time"
)

type Pinger interface {
	Ping(net.Addr) (time.Duration, error)
	Close() error
}
