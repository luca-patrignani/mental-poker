package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// guessIpAddress takes a base IP address and a partial address string,
// and fills in the missing octets from the base address.
func guessIpAddress(baseAddress net.IP, partialAddr string) (net.IP, error) {
	ip := make(net.IP, len(baseAddress))
	copy(ip, baseAddress)
	octets := strings.Split(partialAddr, ".")
	if len(octets) == 1 && octets[0] == "" {
		return ip, nil
	}
	for i := 0; i < len(octets); i++ {
		var octet byte
		_, err := fmt.Sscanf(octets[i], "%d", &octet)
		if err != nil {
			return net.IP{}, err
		}
		ip[len(ip)-len(octets)+i] = octet
	}
	return ip, nil
}

// subnetOfListener returns the IP network (CIDR) of the interface that contains
// the local address used by the provided TCP listener.
func subnetOfListener(l *net.TCPListener) (net.IPNet, error) {
	tcpAddr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		return net.IPNet{}, fmt.Errorf("listener is not TCP")
	}
	ip := tcpAddr.IP
	if ip == nil || ip.IsUnspecified() {
		return net.IPNet{}, fmt.Errorf("listener has unspecified IP %v", ip)
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return net.IPNet{}, err
	}
	for _, ifi := range ifaces {
		addrs, _ := ifi.Addrs()
		for _, a := range addrs {
			var ipnet *net.IPNet
			switch v := a.(type) {
			case *net.IPNet:
				ipnet = v
			case *net.IPAddr:
				ipnet = &net.IPNet{IP: v.IP, Mask: v.IP.DefaultMask()}
			default:
				continue
			}
			if ipnet == nil {
				continue
			}
			if ipnet.Contains(ip) || ipnet.IP.Equal(ip) {
				return *ipnet, nil
			}
		}
	}
	return net.IPNet{}, fmt.Errorf("no interface found for ip %v", ip)
}

// splitHostPort splits an address into host and port, using defaultPort if no port is specified.
func splitHostPort(addr string, defaultPort int) (string, string, error) {
	ipaddr, port, err := net.SplitHostPort(addr)
	if err != nil {
		addr = addr + ":" + strconv.Itoa(defaultPort)
		ipaddr, port, err = net.SplitHostPort(addr)
		if err != nil {
			return "", "", err
		}
	}
	return ipaddr, port, nil
}
