package room

import "sync"

type Manager[T any] struct {
	rooms map[string]*Room[T]
	mu    sync.Mutex
}

func NewManager[T any]() *Manager[T] {
	return &Manager[T]{
		rooms: map[string]*Room[T]{},
	}
}

func (m *Manager[T]) Load(name string, cfg *Config[T]) *Room[T] {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, ok := m.rooms[name]
	if ok {
		return room
	}

	room = New(name, cfg)
	m.rooms[name] = room
	return room
}

func (m *Manager[T]) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var wg sync.WaitGroup
	for _, room := range m.rooms {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Ignore the error
			room.Close()
		}()
	}

	wg.Wait()
	clear(m.rooms)

	return nil
}
