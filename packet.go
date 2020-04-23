package socks5

import (
	"errors"
	"math"
	"net"
	"strconv"
)

/*
	The SOCKS UDP request/response is formed as follows:
	+-----+------+-------+----------+----------+----------+
	| RSV | FRAG |  ATYP | DST.ADDR | DST.PORT |   DATA   |
	+-----+------+-------+----------+----------+----------+
	|  2  |  1   | X'00' | Variable |     2    | Variable |
	+-----+------+-------+----------+----------+----------+
*/
// Packet udp packet
type Packet struct {
	RSV     uint16
	Frag    uint8
	ATYP    uint8
	DstAddr AddrSpec
	Data    []byte
}

func NewEmptyPacket() Packet {
	return Packet{}
}

func NewPacket(destAddr string, data []byte) (p Packet, err error) {
	var host, port string

	host, port, err = net.SplitHostPort(destAddr)
	if err != nil {
		return
	}
	p.DstAddr.Port, err = strconv.Atoi(port)
	if err != nil {
		return
	}
	p.RSV = 0
	p.Frag = 0
	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			p.ATYP = ATYPIPv4
			p.DstAddr.IP = ip4
		} else {
			p.ATYP = ATYPIPV6
			p.DstAddr.IP = ip.To16()
		}
	} else {
		if len(host) > math.MaxUint8 {
			err = errors.New("destination host name too long")
			return
		}
		p.ATYP = ATYPDomain
		p.DstAddr.FQDN = host
	}
	return
}

func (sf *Packet) Parses(b []byte) error {
	if len(b) <= 4+net.IPv4len+2 { // no data
		return errors.New("too short")
	}
	// ignore RSV
	sf.RSV = 0
	// FRAG
	sf.Frag = b[2]
	sf.ATYP = b[3]
	switch sf.ATYP {
	case ATYPIPv4:
		sf.DstAddr.IP = net.IPv4(b[4], b[5], b[6], b[7])
		sf.DstAddr.Port = buildPort(b[4+net.IPv4len], b[4+net.IPv4len+1])
	case ATYPIPV6:
		if len(b) <= (4 + net.IPv6len + 2) {
			return errors.New("too short")
		}

		sf.DstAddr.IP = net.IP{b[4], b[5], b[6], b[7], b[8], b[9], b[10], b[11], b[12], b[13], b[14], b[15], b[16], b[17], b[18], b[19]}
		sf.DstAddr.Port = buildPort(b[4+net.IPv6len], b[4+net.IPv6len+1])
	case ATYPDomain:
		addrLen := int(b[4])
		if len(b) <= (4 + 1 + addrLen + 2) {
			return errors.New("too short")
		}
		str := make([]byte, addrLen)
		copy(str, b[5:5+addrLen])
		sf.DstAddr.FQDN = string(str)
		sf.DstAddr.Port = buildPort(b[5+addrLen], b[5+addrLen+1])
	default:
		return errUnrecognizedAddrType
	}
	return nil
}