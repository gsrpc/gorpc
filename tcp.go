package gorpc

import (
	"io"
	"net"

	"github.com/gsdocker/gserrors"
)

// TCPConnect create a new tcp client
func TCPConnect(builder *ClientBuilder, name, raddr string) (Client, error) {
	return builder.Build(name, func() (io.ReadWriteCloser, error) {
		return net.Dial("tcp", raddr)
	})
}

// TCPListen .
func TCPListen(acceptor *Acceptor, laddr string) error {

	listener, err := net.Listen("tcp", laddr)

	if err != nil {
		return err
	}

	for {
		conn, err := listener.Accept()

		if err != nil {
			return gserrors.Newf(err, "tcp(%s) accept error", laddr)
		}

		acceptor.V("accept connection(%s)", conn.RemoteAddr())

		acceptor.Accept(conn.RemoteAddr().String(), conn)
	}
}
