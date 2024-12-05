package socket

import (
	"fmt"
	"log"
	"slices"

	"github.com/softwarespot/chatterbox/pkg/room"
)

type empty struct{}

type Packet struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

type Socket struct {
	subscribers map[string][]func(args ...any)

	adapter Adapter
	client  *room.Client[Args]

	connected bool

	ackID  int
	ackFns map[int]func(...any)

	disconnectedCh chan empty
}

func New(adapter Adapter) (*Socket, error) {
	s := &Socket{
		subscribers: map[string][]func(args ...any){},

		adapter: adapter,
		client:  nil,

		connected: false,

		ackID:  0,
		ackFns: map[int]func(...any){},

		disconnectedCh: make(chan empty),
	}

	var err error
	if s.client, err = room.NewClient[Args](); err != nil {
		return nil, err
	}

	go s.onPacket()

	return s, nil
}

func (s *Socket) emit(event string, args ...any) {
	for _, fn := range s.subscribers[event] {
		fn(args...)
	}
}

func (s *Socket) on(event string, fn func(args ...any)) {
	s.subscribers[event] = append(s.subscribers[event], fn)
}

func (s *Socket) off(event string, fn func(args ...any)) {
	if event == "" && fn == nil {
		clear(s.subscribers)
		return
	}

	fns, ok := s.subscribers[event]
	if !ok {
		return
	}

	if fn == nil {
		delete(s.subscribers, event)
		return
	}

	s.subscribers[event] = slices.DeleteFunc(fns, func(eventFn func(args ...any)) bool {
		// Compare the func references are the same
		return &fn == &eventFn
	})
}

func (s *Socket) onPacket() {
	for {
		var pkt Packet
		pkt, err := s.adapter.Receive()
		if err != nil {
			s.onDisconnect(err)
			break
		}

		switch pkt.Type {
		case "ack":
			id, ok := pkt.Data["id"].(float64)
			if !ok {
				log.Printf("invalid id type: %v", pkt.Data["id"])
				continue
			}
			ackID := int(id)

			args, ok := pkt.Data["args"].([]any)
			if !ok {
				log.Printf("invalid args type: %v", pkt.Data["args"])
				continue
			}
			if ackFn, ok := s.ackFns[ackID]; ok {
				ackFn(args...)
				delete(s.ackFns, ackID)
			}
		case "event":
			event, ok := pkt.Data["event"].(string)
			if !ok {
				log.Printf("invalid event type: %v", pkt.Data["event"])
				continue
			}

			args, ok := pkt.Data["args"].([]any)
			if !ok {
				log.Printf("invalid args type: %v", pkt.Data["args"])
				continue
			}

			id, ok := pkt.Data["ackId"].(float64)
			if !ok {
				log.Printf("invalid ackId type: %v", pkt.Data["ackId"])
				continue
			}
			ackID := int(id)

			if ackID > 0 {
				args = append(args, func(args ...any) {
					s.emitAck(ackID, args...)
				})
			}
			s.emit(event, args...)
		}
	}
}

func (s *Socket) onConnect() error {
	err := s.adapter.Send(Packet{
		Type: "connect",
		Data: map[string]any{
			"id": s.ID(),
		},
	})
	if err != nil {
		return fmt.Errorf("socket: sending connect packet: %w", err)
	}

	s.connected = true

	s.emit("connect", s.ID())

	<-s.disconnectedCh
	return nil
}

func (s *Socket) onDisconnect(err error) error {
	defer close(s.disconnectedCh)

	if !s.connected {
		return nil
	}

	reason := err.Error()
	err = s.adapter.Send(Packet{
		Type: "disconnect",
		Data: map[string]any{
			"reason": reason,
		},
	})
	if err != nil {
		return fmt.Errorf("socket: sending disconnect packet: %w", err)
	}

	s.connected = false
	s.client = nil

	s.ackID = 0
	clear(s.ackFns)

	s.emit("disconnect", reason)

	if err := s.adapter.Close(); err != nil {
		return fmt.Errorf("socket: closing websocket connection: %w", err)
	}
	return nil
}

func (s *Socket) ID() string {
	if s.client == nil {
		return ""
	}
	return s.client.ID()
}

func (s *Socket) Client() *room.Client[Args] {
	return s.client
}

func (s *Socket) Connected() bool {
	return s.connected
}

func (s *Socket) Disconnected() bool {
	return !s.connected
}

func (s *Socket) Emit(event string, args ...any) error {
	var ackID int
	if ackFn, ok := GetAckFunc(args); ok {
		s.ackID++
		s.ackFns[s.ackID] = ackFn

		// Remove the "ack" function
		args = argDeleteLast(args)
		ackID = s.ackID
	}

	err := s.adapter.Send(Packet{
		Type: "event",
		Data: map[string]any{
			"event": event,
			"args":  ensureNonEmptyArgs(args),
			"ackId": ackID,
		},
	})
	if err != nil {
		return fmt.Errorf("socket: calling emit: %w", err)
	}
	return nil
}

func (s *Socket) emitAck(id int, args ...any) error {
	err := s.adapter.Send(Packet{
		Type: "ack",
		Data: map[string]any{
			"id":   id,
			"args": ensureNonEmptyArgs(args),
		},
	})
	if err != nil {
		return fmt.Errorf("socket: calling emitAck: %w", err)
	}
	return nil
}

func (s *Socket) On(event string, fn func(args ...any)) *Socket {
	s.on(event, fn)
	return s
}

func (s *Socket) Off(event string, fn func(args ...any)) *Socket {
	s.off(event, fn)
	return s
}

func ensureNonEmptyArgs(args []any) []any {
	if len(args) == 0 {
		return []any{}
	}
	return args
}
