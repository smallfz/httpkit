package ws

import (
	"net"
)

type RemoteAddresser interface {
	RemoteAddr() net.Addr
}
