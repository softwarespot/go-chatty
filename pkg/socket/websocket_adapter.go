package socket

import (
	"errors"
	"fmt"
	"io"

	"golang.org/x/net/websocket"
)

type WebSocketAdapter struct {
	conn *websocket.Conn
}

func NewWebSocketAdapter(conn *websocket.Conn) *WebSocketAdapter {
	return &WebSocketAdapter{
		conn: conn,
	}
}

func (w *WebSocketAdapter) Receive() (Packet, error) {
	var pkt Packet
	if err := websocket.JSON.Receive(w.conn, &pkt); err != nil {
		if errors.Is(err, io.EOF) {
			return Packet{}, errors.New("socket: client unreachable when receiving")
		}
		return Packet{}, fmt.Errorf("socket: client unreachable when receiving with error: %w", err)
	}
	return pkt, nil
}

func (w *WebSocketAdapter) Send(pkt Packet) error {
	if err := websocket.JSON.Send(w.conn, pkt); err != nil {
		return fmt.Errorf("socket: client unreachable when sending with error: %w", err)
	}
	return nil
}

func (w *WebSocketAdapter) Close() error {
	if err := w.conn.Close(); err != nil {
		return fmt.Errorf("socket: closing with error: %s", err.Error())
	}
	return nil
}
