package addr

import (
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// ICMPQueryAddr represents the address of a ICMP query end point.
type ICMPQueryAddr struct {
	IP net.IP
	Id uint16
}

func (addr ICMPQueryAddr) String() string {
	return fmt.Sprintf("%s@%d", formatIP(addr.IP), addr.Id)
}

func (addr ICMPQueryAddr) Network() string {
	return "icmp query"
}

type MultiIPAddr struct {
	IPs []net.IP
}

func (addr MultiIPAddr) String() string {
	s := make([]string, 0)

	for _, ip := range addr.IPs {
		a := net.IPAddr{IP: ip}
		s = append(s, a.String())
	}

	return strings.Join(s, ", ")
}

func (addr MultiIPAddr) Network() string {
	return "ip"
}

func (addr *MultiIPAddr) IPv4() net.IP {
	for _, ip := range addr.IPs {
		if ip.To4() != nil {
			return ip
		}
	}

	return nil
}

func (addr *MultiIPAddr) IPv6() net.IP {
	for _, ip := range addr.IPs {
		if ip.To4() == nil {
			return ip
		}
	}

	return nil
}

// ParseTCPAddr returns an TCPAddr by the given address
func ParseTCPAddr(s string) (*net.TCPAddr, error) {
	ipStr, portStr, err := net.SplitHostPort(s)
	if err != nil {
		return nil, fmt.Errorf("split host port: %w", err)
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid ip %s", ipStr)
	}

	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("parse port %s: %w", portStr, err)
	}

	return &net.TCPAddr{IP: ip, Port: int(port)}, nil
}

// ParseAddr returns an Addr by the given address
func ParseAddr(s string) (net.Addr, error) {
	// Guess port
	if s[0] == ':' {
		port, err := strconv.ParseUint(s[1:], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("parse port %s: %w", s[1:], err)
		}
		return &net.TCPAddr{
			IP:   nil,
			Port: int(port),
		}, nil
	}

	// Guess IP and port
	addr, err := ParseTCPAddr(s)
	if err != nil {
		// IP
		ip := net.ParseIP(s)
		if ip == nil {
			return nil, fmt.Errorf("invalid ip %s", s)
		}
		return &net.IPAddr{IP: ip}, nil
	}

	return addr, nil
}

func formatIP(ip net.IP) string {
	if ip == nil {
		return ""
	}

	if ip.To4() != nil {
		return ip.String()
	} else {
		return fmt.Sprintf("[%s]", ip)
	}
}

func fullString(ip net.IP) string {
	if ip.To4() != nil {
		return ip.String()
	}
	dst := make([]byte, hex.EncodedLen(len(ip)))
	_ = hex.Encode(dst, ip)
	return string(dst[0:4]) + ":" +
		string(dst[4:8]) + ":" +
		string(dst[8:12]) + ":" +
		string(dst[12:16]) + ":" +
		string(dst[16:20]) + ":" +
		string(dst[20:24]) + ":" +
		string(dst[24:28]) + ":" +
		string(dst[28:])
}

func bpfFilter(prefix string, addr net.Addr) (string, error) {
	switch t := addr.(type) {
	case *net.IPAddr:
		return fmt.Sprintf("(%s host %s)", prefix, fullString(addr.(*net.IPAddr).IP)), nil
	case *net.TCPAddr:
		tcpAddr := addr.(*net.TCPAddr)

		if tcpAddr.IP == nil {
			return fmt.Sprintf("(%s port %d)", prefix, tcpAddr.Port), nil
		} else {
			return fmt.Sprintf("(%s host %s && %s port %d)", prefix, fullString(tcpAddr.IP), prefix, tcpAddr.Port), nil
		}
	default:
		return "", fmt.Errorf("type %T not support", t)
	}
}

// SrcBPFFilter returns a source BPF filter by the giver address
func SrcBPFFilter(addr net.Addr) (string, error) {
	return bpfFilter("src", addr)
}

// DstBPFFilter returns a destination BPF filter by the giver address
func DstBPFFilter(addr net.Addr) (string, error) {
	return bpfFilter("dst", addr)
}