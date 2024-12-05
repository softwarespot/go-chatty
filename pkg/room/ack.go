package room

import "sync"

type ack struct {
	closeOnce sync.Once
	closed    bool
	ch        chan error
}

func newACK() *ack {
	return &ack{
		ch: make(chan error),
	}
}

func (a *ack) done(err error) {
	a.ch <- err
}

func (a *ack) waiter() <-chan error {
	return a.ch
}

func (a *ack) wait() error {
	err := <-a.waiter()
	a.close()
	return err
}

func (a *ack) close() {
	a.closeOnce.Do(func() {
		close(a.ch)
		a.closed = true
	})
}
