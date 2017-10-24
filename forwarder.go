package main

import "log"

type forwarder struct {
	join    chan *client
	leave   chan *client
	forward chan *Tweet
	clients map[*client]bool
}

func newForwarder() *forwarder {
	return &forwarder{
		join:    make(chan *client),
		leave:   make(chan *client),
		forward: make(chan *Tweet),
		clients: make(map[*client]bool),
	}
}

func (f *forwarder) run() {
	for {
		select {
		case client := <-f.join:
			log.Print("- New client joined")
			f.clients[client] = true
		case client := <-f.leave:
			log.Print("- Client left")
			delete(f.clients, client)
			close(client.send)
		case tweet := <-f.forward:
			log.Print("- New cat photo received")
			for client := range f.clients {
				client.send <- tweet.GetPhotoURL()
			}
		}
	}
}
