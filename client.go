package main

import "github.com/gorilla/websocket"

type client struct {
	send   chan string
	socket *websocket.Conn
}

func newClient(socket *websocket.Conn) *client {
	return &client{
		send:   make(chan string),
		socket: socket,
	}
}

func (c *client) write() {
	defer c.socket.Close()
	for tweet := range c.send {
		err := c.socket.WriteMessage(websocket.TextMessage, []byte(tweet))
		if err != nil {
			return
		}
	}
}
