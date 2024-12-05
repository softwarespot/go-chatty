package room

import (
	"errors"
	"sync"
)

var ErrClientClosed = errors.New("room: client is closed")

type Client[T any] struct {
	id     string
	closed bool

	msgCh chan T
	mu    sync.Mutex
}

func NewClient[T any]() (*Client[T], error) {
	c := &Client[T]{
		id:     "",
		closed: false,
		msgCh:  make(chan T),
	}

	var err error
	if c.id, err = createID(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client[T]) ID() string {
	return c.id
}

func (c *Client[T]) Send(msg T) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrClientClosed
	}

	c.msgCh <- msg
	return nil
}

func (c *Client[T]) Messages() <-chan T {
	return c.msgCh
}

func (c *Client[T]) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrClientClosed
	}

	c.closed = true
	close(c.msgCh)

	return nil
}
