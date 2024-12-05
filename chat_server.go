package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/softwarespot/chatterbox/pkg/room"
	"github.com/softwarespot/chatterbox/pkg/socket"
	"golang.org/x/net/websocket"
)

type ChatServer struct {
	server *websocket.Server
	rm     *room.Manager[socket.Args]
}

func NewChatServer() *ChatServer {
	cs := &ChatServer{
		server: nil,
		rm:     room.NewManager[socket.Args](),
	}

	cs.server = &websocket.Server{
		Config: websocket.Config{
			// Allow any origin
			Origin: nil,
		},
		Handshake: func(_ *websocket.Config, _ *http.Request) error {
			// Allow any origin
			return nil
		},
		Handler: cs.ServeChat,
	}
	return cs
}

func (cs *ChatServer) ServeChat(conn *websocket.Conn) {
	r := conn.Request()
	log.Printf("connection established for %s", r.RemoteAddr)
	defer log.Printf("connection disconnected for %s", r.RemoteAddr)

	err := socket.IO(conn, func(s *socket.Socket) error {
		c := s.Client()
		var currRoom *room.Room[socket.Args]

		leaveRoomFn := func() {
			if currRoom == nil {
				return
			}

			currRoom.Unregister(c)

			c.Send(socket.Args{
				"System",
				fmt.Sprintf("Left the room %s.", currRoom.Name()),
			})
			currRoom.Send(c, socket.Args{
				"System",
				fmt.Sprintf("Socket ID %s left the room %s.", c.ID(), currRoom.Name()),
			})

			log.Printf("socket ID %s left the room %s", c.ID(), currRoom.Name())

			currRoom = nil
		}

		s.On("connect", func(_ ...any) {
			log.Printf("socket ID %s opened the connection", c.ID())

			go func() {
				for m := range c.Messages() {
					if err := s.Emit("message", m...); err != nil {
						log.Printf("error sending message to socket ID %s: %v", c.ID(), err)
						break
					}
				}
			}()
		}).On("disconnect", func(_ ...any) {
			log.Printf("socket ID %s closed the connection", c.ID())

			leaveRoomFn()
		})

		s.On("ping", func(args ...any) {
			if ackFn, ok := socket.GetAckFunc(args); ok {
				ackFn()
			}
		})

		s.On("join", func(args ...any) {
			leaveRoomFn()

			roomName, err := socket.ArgAt[string](args, 0)
			if err != nil {
				log.Printf("socket ID %s encountered error: %v", c.ID(), err)
				return
			}

			currRoom = cs.rm.Load(roomName, nil)
			log.Printf("socket ID %s loaded the room %s", c.ID(), currRoom.Name())

			currRoom.Register(c)

			c.Send(socket.Args{
				"System",
				fmt.Sprintf("Joined the room %s. Currently there are %d client(s).", currRoom.Name(), currRoom.Size()-1),
			})
			currRoom.Send(c, socket.Args{
				"System",
				fmt.Sprintf("Socket ID %s joined the room %s.", c.ID(), currRoom.Name()),
			})

			ackFn, ok := socket.GetAckFunc(args)
			if ok {
				ackFn()
			}

			log.Printf("socket ID %s joined the room %s", c.ID(), currRoom.Name())
		})

		s.On("leave", func(_ ...any) {
			leaveRoomFn()
		})

		s.On("message", func(args ...any) {
			if currRoom == nil {
				return
			}

			msg, err := socket.ArgAt[string](args, 0)
			if err != nil {
				log.Printf("socket ID %s encountered error: %v", c.ID(), err)
				return
			}

			c.Send(socket.Args{
				"Sender",
				msg,
			})
			currRoom.Send(c, socket.Args{
				"Receiver",
				msg,
			})
			log.Printf("socket ID %s broadcast message %q to the room %s", c.ID(), msg, currRoom.Name())
		})

		return nil
	})

	log.Printf("socket completed: %v", err)
}

func (cs *ChatServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cs.server.ServeHTTP(w, r)
}
