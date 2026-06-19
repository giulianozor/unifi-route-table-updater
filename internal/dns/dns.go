package dns

import (
	"fmt"
	"net"
)

func ResolveIPv4(name string) (string, error) {
	addrs, err := net.LookupHost(name)
	if err != nil {
		return "", err
	}
	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip != nil && ip.To4() != nil {
			return ip.String(), nil
		}
	}
	return "", fmt.Errorf("no IPv4 addresses found for %s", name)
}
