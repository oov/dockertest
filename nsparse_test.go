package dockertest

import (
	"net"
	"testing"
)

func TestParseProcNet(t *testing.T) {
	tests := []struct {
		Input   string
		Entries []entry
	}{
		{
			Input: `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode                                                     
   0: 00000000:1538 00000000:0000 0A 00000000:00000000 00:00000000 00000000   999        0 2575988 1 0000000000000000 100 0 0 10 0                   `,
			Entries: []entry{
				entry{
					Number:     0,
					LocalAddr:  net.IP([]byte{0, 0, 0, 0}),
					LocalPort:  5432,
					RemoteAddr: net.IP([]byte{0, 0, 0, 0}),
					RemotePort: 0,
					UID:        999,
					INode:      2575988,
				},
			},
		},
		{
			Input: `  sl  local_address                         remote_address                        st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 00000000000000000000000000000000:1538 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000   999        0 2575989 1 0000000000000000 100 0 0 10 -1`,
			Entries: []entry{
				entry{
					Number:     0,
					LocalAddr:  net.IP([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}),
					LocalPort:  5432,
					RemoteAddr: net.IP([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}),
					RemotePort: 0,
					UID:        999,
					INode:      2575989,
				},
			},
		},
	}
	for i, tt := range tests {
		es, err := parseProcNet(tt.Input)
		if err != nil {
			t.Errorf("tests[%d]: %v", i, err)
		}
		if len(es) != len(tt.Entries) {
			t.Errorf("tests[%d]: len(es) want %q got %q", i, len(tt.Entries), len(es))
		}
		for j, e := range es {
			g := tt.Entries[j]
			if e.Number != g.Number {
				t.Errorf("tests[%d] entry[%d] e.Number: want %q got %q", i, j, g.Number, e.Number)
			}
			if e.LocalAddr.String() != g.LocalAddr.String() {
				t.Errorf("tests[%d] entry[%d] e.LocalAddr: want %q got %q", i, j, g.LocalAddr, e.LocalAddr)
			}
			if e.LocalPort != g.LocalPort {
				t.Errorf("tests[%d] entry[%d] e.LocalPort: want %q got %q", i, j, g.LocalPort, e.LocalPort)
			}
			if e.RemoteAddr.String() != g.RemoteAddr.String() {
				t.Errorf("tests[%d] entry[%d] e.RemoteAddr: want %q got %q", i, j, g.RemoteAddr, e.RemoteAddr)
			}
			if e.RemotePort != g.RemotePort {
				t.Errorf("tests[%d] entry[%d] e.RemotePort: want %q got %q", i, j, g.RemotePort, e.RemotePort)
			}
			if e.UID != g.UID {
				t.Errorf("tests[%d] entry[%d] e.UID: want %d got %d", i, j, g.UID, e.UID)
			}
			if e.INode != g.INode {
				t.Errorf("tests[%d] entry[%d] e.INode: want %d got %d", i, j, g.INode, e.INode)
			}
		}
	}
}
