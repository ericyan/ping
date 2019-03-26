package iputil

import (
	"errors"
	"net"
)

// InterfaceAddr represents an interface IP address and the name of its
// associated network interface.
type InterfaceAddr struct {
	*net.IPNet
	InterfaceName string
}

// Interface returns the associated network interface.
func (addr *InterfaceAddr) Interface() (*net.Interface, error) {
	return net.InterfaceByName(addr.InterfaceName)
}

// InterfaceAddrs returns a list of the system's unicast interface IP
// addresses.
func InterfaceAddrs() ([]*InterfaceAddr, error) {
	var ifAddrs []*InterfaceAddr

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			ifAddrs = append(ifAddrs, &InterfaceAddr{addr.(*net.IPNet), iface.Name})
		}
	}

	return ifAddrs, nil
}

// DefaultIPv4 returns the first non-loopback interface IPv4 address.
func DefaultIPv4() (*InterfaceAddr, error) {
	ifAddrs, err := InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range ifAddrs {
		if IsIPv4(addr.IP) && !addr.IP.IsLoopback() {
			return addr, nil
		}
	}

	return nil, errors.New("default ipv4 unavailable")
}

// DefaultIPv6 returns the first non-loopback interface IPv6 address.
func DefaultIPv6() (*InterfaceAddr, error) {
	ifAddrs, err := InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range ifAddrs {
		if IsIPv6(addr.IP) && !IsIPv4(addr.IP) && !addr.IP.IsLoopback() {
			return addr, nil
		}
	}

	return nil, errors.New("default ipv6 unavailable")
}
