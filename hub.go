package main

type byteArr []byte
type Hub struct {
	clients map[*Client]bool
	broadcast chan struct {*Client; byteArr}
	register chan *Client
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub {
		clients: make(map[*Client]bool),
		broadcast: make(chan struct {*Client; byteArr}),
		register: make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub)run() {
	for {
		select {
		case c := <-h.register:
			h.clients[c] = true
		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
		case msg := <- h.broadcast:
			for c := range h.clients {
				select {
				case c.send <- struct {byteArr; bool}{msg.byteArr, c == msg.Client}:
				default:
					close(c.send)
					delete(h.clients, c)
				}
			}
		}
	}
}
