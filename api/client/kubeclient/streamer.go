package kubeclient

import (
	"net"
	"time"

	"github.com/gorilla/websocket"
)

func newWebsocketStreamer(conn *websocket.Conn, done chan struct{}) *wsStreamer {
	return &wsStreamer{
		conn: conn,
		done: done,
	}
}

type wsStreamer struct {
	conn *websocket.Conn
	done chan struct{}
}

func (ws *wsStreamer) streamDone() {
	close(ws.done)
}

func (ws *wsStreamer) Stream(options StreamOptions) error {
	copyErr := make(chan error, 1)

	go func() {
		_, err := CopyTo(ws.conn, options.In)
		copyErr <- err
	}()

	go func() {
		_, err := CopyFrom(options.Out, ws.conn)
		copyErr <- err
	}()

	defer ws.streamDone()
	return <-copyErr
}

func (ws *wsStreamer) AsConn() net.Conn {
	return &wsConn{
		Conn:         ws.conn,
		binaryReader: &binaryReader{conn: ws.conn},
		binaryWriter: &binaryWriter{conn: ws.conn},
	}
}

type wsConn struct {
	*websocket.Conn
	*binaryReader
	*binaryWriter
}

func (c *wsConn) SetDeadline(t time.Time) error {
	if err := c.Conn.SetWriteDeadline(t); err != nil {
		return err
	}
	return c.Conn.SetReadDeadline(t)
}
