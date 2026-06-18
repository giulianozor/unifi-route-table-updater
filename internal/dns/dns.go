package dns

import (
	"net"
	"strings"
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
		if strings.Contains(addr, ".") {
			return addr, nil
		}
	}
	if len(addrs) > 0 {
		return addrs[0], nil
	}
	return "", nil
}
