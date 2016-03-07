package gorpc

import (
	"io"
	"net/url"

	"github.com/gorilla/websocket"
)

// WebSocketConnect establish a websocket client channel
func WebSocketConnect(builder *ClientBuilder, name string, u *url.URL) (Client, error) {
	return builder.Build(name, func() (io.ReadWriteCloser, error) {

		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)

		if err != nil {
			return nil, err
		}

		return c.UnderlyingConn(), nil
	})
}
