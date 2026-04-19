package udp

import (
	"github.com/pysugar/netool/buf"
	"github.com/pysugar/netool/net"
)

// Packet is a UDP packet together with its source and destination address.
type Packet struct {
	Payload *buf.Buffer
	Source  net.Destination
	Target  net.Destination
}
