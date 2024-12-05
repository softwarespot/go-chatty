package room

import (
	"fmt"
	"testing"
	"time"

	testhelpers "github.com/softwarespot/chatterbox/pkg/test-helpers"
)

func Test_NewRoom(t *testing.T) {
	cfg := NewRoomConfig[string]()
	cfg.CloseTimeout = 5 * time.Second

	r := New("root", cfg)

	c1, err := NewClient[string]()
	testhelpers.AssertNoError(t, err)

	c2, err := NewClient[string]()
	testhelpers.AssertNoError(t, err)

	testhelpers.AssertNoError(t, r.Register(c1))
	testhelpers.AssertNoError(t, r.Register(c2))

	go func() {
		for msg := range c1.Messages() {
			t.Log("client 1:", msg)
		}
		r.Unregister(c1)
		t.Log("client 1 unregistered")
	}()
	go func() {
		for msg := range c2.Messages() {
			t.Log("client 2:", msg)
		}
		r.Unregister(c2)
		t.Log("client 2 unregistered")
	}()

	testhelpers.AssertEqual(t, r.Name(), "root")

	go func() {
		i := 0
		for {
			if err := r.Send(c1, fmt.Sprintf("from client %s (%d)", c1.ID(), i)); err != nil {
				break
			}
			i += 1
		}
	}()

	testhelpers.AssertNoError(t, r.Broadcast("for ALL clients"))

	time.Sleep(1 * time.Millisecond)

	testhelpers.AssertNoError(t, r.Close())
	testhelpers.AssertError(t, r.Broadcast(""))
	testhelpers.AssertError(t, r.Send(c1, ""))
	testhelpers.AssertEqual(t, r.Size(), 0)

	time.Sleep(1 * time.Millisecond)
}
