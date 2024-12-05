package socket

import (
	"fmt"

	"golang.org/x/net/websocket"
)

func IO(conn *websocket.Conn, initFn func(s *Socket) error) error {
	s, err := New(NewWebSocketAdapter(conn))
	if err != nil {
		return fmt.Errorf("socket: initializing socket: %w", err)
	}
	if err := initFn(s); err != nil {
		return fmt.Errorf("socket: initializing socket with the initialization function: %w", err)
	}
	if err := s.onConnect(); err != nil {
		return err
	}
	return nil
}
