package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan struct {byteArr; bool}
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space = []byte{' '}
)

func (c *Client) read() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		return
	}
	c.conn.SetPongHandler(
		func(string) error {
			return c.conn.SetReadDeadline(time.Now().Add(pongWait))
		})
	for {
		_, m, err := c.conn.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}
		m = bytes.TrimSpace(bytes.Replace(m, newline, space, -1))
		var data map[string]interface{}
		if err := json.Unmarshal(m, &data); err != nil {
			log.Println(err)
			return
		}
		if msg, ok := data["msg"].(string); ok {
			c.hub.broadcast <- struct {*Client; byteArr}{c, []byte(msg)}
		}
	}
}

func (c *Client) write() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case m, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if !ok {
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					return
				}
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if msg(m).Render(context.Background(), w) != nil {
				return
			}
			n := len(c.send)
			for i := 0; i < n; i++ {
				if msg(<-c.send).Render(context.Background(), w) != nil {
					return
				}
			}
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func serveWs(h *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	c := &Client{hub: h, conn: conn, send: make(chan struct{byteArr; bool})}
	c.hub.register <- c
	go c.write()
	go c.read()
}
