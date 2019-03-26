package iputil

import (
	"net"

	"github.com/ericyan/iputil/internal/uint128"
)

// NetworkAddr returns the network address, which is also the beginning
// address, of the subnet.
func NetworkAddr(subnet *net.IPNet) net.IP {
	return subnet.IP
}

// BroadcastAddr returns the broadcast address, which is also the ending
// address, of the subnet.
func BroadcastAddr(subnet *net.IPNet) net.IP {
	n := len(subnet.IP)
	if n != len(subnet.Mask) {
		return nil
	}

	addr := make(net.IP, n)
	for i := 0; i < n; i++ {
		addr[i] = subnet.IP[i] | ^subnet.Mask[i]
	}

	return addr
}

// Subnets divides the supernet into smaller subnets of given prefix
// size. It returns nil if subnet prefix size is invalid.
func Subnets(supernet *net.IPNet, prefix int) []*net.IPNet {
	ones, bits := supernet.Mask.Size()
	if ones > prefix || bits < prefix {
		return nil
	}

	ip := supernet.IP
	mask := net.CIDRMask(prefix, bits)
	size, _ := uint128.Pow2(uint(bits - prefix))

	subnets := make([]*net.IPNet, 1<<uint(prefix-ones))
	for i := 0; i < len(subnets); i++ {
		if i > 0 {
			last, _ := uint128.NewFromBytes(subnets[i-1].IP)
			buf := last.Add(size).Bytes()

			// Uint128 always returns a 16-byte slice. We only need the last
			// 4 bytes for IPv4 addresses.
			ip = buf[16-len(ip):]
		}

		subnets[i] = &net.IPNet{
			IP:   ip,
			Mask: mask,
		}
	}

	return subnets
}
