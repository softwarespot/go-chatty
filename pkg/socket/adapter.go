package socket

type Adapter interface {
	Receive() (Packet, error)
	Send(Packet) error
	Close() error
}
