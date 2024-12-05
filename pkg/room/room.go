package room

import (
	"errors"
	"sync/atomic"
	"time"
)

var (
	ErrRoomClosed       = errors.New("room: room is closed")
	ErrRoomCloseTimeout = errors.New("room: timeout waiting for the clients to close")
	ErrRoomClientNil    = errors.New("room: client cannot be nil")
)

type empty struct{}

type roomRegistration[T any] struct {
	client *Client[T]
	ack    *ack
}

type roomMessage[T any] struct {
	msg T

	// Sender is optional.
	// When not defined with a client, the message is sent to ALL clients
	sender *Client[T]

	// Acknowledgement is optional
	ack *ack
}

type roomClose struct {
	ack *ack
}

type Room[T any] struct {
	name string
	cfg  *Config[T]

	closed       atomic.Bool
	closeCh      chan roomClose
	registerCh   chan roomRegistration[T]
	unregisterCh chan roomRegistration[T]

	clients map[*Client[T]]empty
	size    atomic.Int64

	msgCh chan roomMessage[T]
}

func New[T any](name string, cfg *Config[T]) *Room[T] {
	if cfg == nil {
		cfg = NewRoomConfig[T]()
	}
	r := &Room[T]{
		name: name,
		cfg:  cfg,

		closeCh:      make(chan roomClose),
		registerCh:   make(chan roomRegistration[T]),
		unregisterCh: make(chan roomRegistration[T]),

		clients: map[*Client[T]]empty{},

		msgCh: make(chan roomMessage[T]),
	}

	go r.start()

	return r
}

func (r *Room[T]) start() {
handler:
	for {
		select {
		case rc := <-r.closeCh:
			r.closed.Store(true)
			r.drainClose()
			r.clientsClose()
			r.storeSize()

			rc.ack.done(nil)
			break handler
		case rr := <-r.registerCh:
			r.clients[rr.client] = empty{}
			r.storeSize()

			rr.ack.done(nil)
		case rr := <-r.unregisterCh:
			delete(r.clients, rr.client)
			r.storeSize()

			rr.ack.done(nil)
		case rm := <-r.msgCh:
			for client := range r.clients {
				if rm.sender == client {
					continue
				}

				// Ignore the error currently
				client.Send(rm.msg)
			}
			rm.ack.done(nil)
		}
	}
}

func (r *Room[T]) Name() string {
	return r.name
}

func (r *Room[T]) Size() int {
	return int(r.size.Load())
}

func (r *Room[T]) storeSize() {
	size := int64(len(r.clients))
	r.size.Store(size)
}

func (r *Room[T]) Register(client *Client[T]) error {
	if client == nil {
		return ErrRoomClientNil
	}
	if r.closed.Load() {
		return ErrRoomClosed
	}

	req := roomRegistration[T]{
		client: client,
		ack:    newACK(),
	}
	r.registerCh <- req

	return req.ack.wait()
}

func (r *Room[T]) Unregister(client *Client[T]) error {
	if client == nil {
		return ErrRoomClientNil
	}
	if r.closed.Load() {
		return ErrRoomClosed
	}

	req := roomRegistration[T]{
		client: client,
		ack:    newACK(),
	}
	r.unregisterCh <- req

	return req.ack.wait()
}

func (r *Room[T]) Send(sender *Client[T], msg T) error {
	return r.send(sender, msg)
}

func (r *Room[T]) Broadcast(msg T) error {
	return r.send(nil, msg)
}

func (r *Room[T]) send(sender *Client[T], msg T) error {
	if r.closed.Load() {
		return ErrRoomClosed
	}

	req := roomMessage[T]{
		msg:    msg,
		sender: sender,
		ack:    newACK(),
	}
	r.msgCh <- req

	return req.ack.wait()
}

func (r *Room[T]) Close() error {
	if r.closed.Load() {
		return ErrRoomClosed
	}

	req := roomClose{
		ack: newACK(),
	}
	r.closeCh <- req

	// Wait for all the clients to close or on timeout
	select {
	case err := <-req.ack.waiter():
		req.ack.close()
		return err
	case <-time.After(r.cfg.CloseTimeout):
		return ErrRoomCloseTimeout
	}
}

func (r *Room[T]) drainClose() {
	ticker := time.NewTicker(256 * time.Millisecond)
	defer ticker.Stop()

	hasBeenCalled := false
drainer:
	for {
		select {
		case rc := <-r.closeCh:
			hasBeenCalled = true
			rc.ack.done(ErrClientClosed)
		case rr := <-r.registerCh:
			hasBeenCalled = true
			rr.ack.done(ErrClientClosed)
		case rr := <-r.unregisterCh:
			hasBeenCalled = true
			rr.ack.done(ErrClientClosed)
		case rm := <-r.msgCh:
			hasBeenCalled = true
			rm.ack.done(ErrClientClosed)
		default:
			select {
			case <-ticker.C:
				// If one of the channels has been called, then wait again for the next tick to ensure
				// no channels have been called, and they have all drained
				if hasBeenCalled {
					hasBeenCalled = false
					continue drainer
				}

				close(r.closeCh)
				close(r.registerCh)
				close(r.unregisterCh)
				close(r.msgCh)
				break drainer
			default:
			}
		}
	}
}

func (r *Room[T]) clientsClose() {
	for client := range r.clients {
		// Ignore the error
		client.Close()
	}
	clear(r.clients)
}
