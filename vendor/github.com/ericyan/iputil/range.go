package iputil

import (
	"errors"
	"net"

	"github.com/ericyan/iputil/internal/uint128"
)

// A Range represents an arbitrary IP address range.
type Range struct {
	af    uint
	first uint128.Int
	last  uint128.Int
}

// NewRange returns a new Range.
func NewRange(first, last net.IP) (*Range, error) {
	if AddressFamily(first) != AddressFamily(last) {
		return nil, errors.New("invalid range")
	}
	r := &Range{af: AddressFamily(first)}

	r.first, _ = uint128.NewFromBytes(first)
	r.last, _ = uint128.NewFromBytes(last)
	if !r.first.IsLessThan(r.last) {
		return nil, errors.New("invalid range")
	}

	return r, nil
}

// First returns the first IP address within the range.
func (r *Range) First() net.IP {
	byteLen := 4
	if r.af == IPv6 {
		byteLen = 16
	}

	return r.first.Bytes()[16-byteLen:]
}

// Last returns the last IP address within the range.
func (r *Range) Last() net.IP {
	byteLen := 4
	if r.af == IPv6 {
		byteLen = 16
	}

	return r.last.Bytes()[16-byteLen:]
}

// Contains reports whether the range includes ip.
func (r *Range) Contains(ip net.IP) bool {
	x, _ := uint128.NewFromBytes(ip)

	if x.IsLessThan(r.first) || x.IsGreaterThan(r.last) {
		return false
	}

	return true
}

// CIDR returns CIDR notation(s) for the range.
func (r *Range) CIDR() []*net.IPNet {
	results := make([]*net.IPNet, 0)

	var maxPrefix int
	switch r.af {
	case IPv4:
		maxPrefix = IPv4BitLen
	case IPv6:
		maxPrefix = IPv6BitLen
	}

	cur := r.first
	for cur.IsLessThan(r.last) || cur.IsEqualTo(r.last) {
		// Number of zeros in netmask to represent current IP.
		zeros := cur.TrailingZeros()

		// Number of remaining IPs.
		n := r.last.Sub(cur).Add(uint128.One)

		// log2n is the max k that 2**k < n. This ensures cidrLen < n.
		if log2n := 128 - n.LeadingZeros() - 1; zeros > log2n {
			zeros = log2n
		}

		results = append(results, &net.IPNet{
			IP:   cur.Bytes()[16-maxPrefix/8:],
			Mask: net.CIDRMask(maxPrefix-zeros, maxPrefix),
		})

		cidrLen, _ := uint128.Pow2(uint(zeros))
		cur = cur.Add(cidrLen)
	}

	return results
}

// Network returns the network name, "ip+net".
func (r *Range) Network() string {
	return "ip+net"
}

// String returns the string form of range.
func (r *Range) String() string {
	return r.First().String() + " - " + r.Last().String()
}
