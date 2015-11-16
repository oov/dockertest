package dockertest

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

type entry struct {
	Number     int
	LocalAddr  net.IP
	LocalPort  int
	RemoteAddr net.IP
	RemotePort int
	UID        int
	INode      int
}

type entries []entry

func (es entries) FindByLocalPort(port int) int {
	for i, e := range es {
		if e.LocalPort == port {
			return i
		}
	}
	return -1
}

func (es entries) FindByRemotePort(port int) int {
	for i, e := range es {
		if e.RemotePort == port {
			return i
		}
	}
	return -1
}

func parseAddr(addr string) (ip net.IP, port int, err error) {
	if len(addr) == 13 {
		ip = make([]byte, 4, 4)
		_, err = fmt.Sscanf(
			addr,
			"%02x%02x%02x%02x:%04x",
			&ip[0], &ip[1], &ip[2], &ip[3],
			&port,
		)
		return
	}
	ip = make([]byte, 16, 16)
	_, err = fmt.Sscanf(
		addr,
		"%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x:%04x",
		&ip[0], &ip[1], &ip[2], &ip[3], &ip[4], &ip[5], &ip[6], &ip[7],
		&ip[8], &ip[9], &ip[10], &ip[11], &ip[12], &ip[13], &ip[14], &ip[15],
		&port,
	)
	return
}

func parseProcNet(s string) (entries, error) {
	var es []entry
	sc := bufio.NewScanner(strings.NewReader(s))
	sc.Scan() // ignore header
	for sc.Scan() {
		var e entry
		var local, remote, skip string
		_, err := fmt.Sscanf(
			sc.Text(),
			"%d: %s %s %s %s %s 00000000 %d 0 %d",
			&e.Number,
			&local,
			&remote,
			&skip,
			&skip,
			&skip,
			&e.UID,
			&e.INode,
		)
		if err != nil {
			return nil, err
		}
		e.LocalAddr, e.LocalPort, err = parseAddr(local)
		if err != nil {
			return nil, err
		}
		e.RemoteAddr, e.RemotePort, err = parseAddr(remote)
		if err != nil {
			return nil, err
		}
		es = append(es, e)
	}
	if sc.Err() != nil {
		return nil, sc.Err()
	}
	return entries(es), nil
}
