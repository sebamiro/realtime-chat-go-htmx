package main

import (
	"context"
	"log"
	"net/http"
)

var hub = newHub()

func handler(w http.ResponseWriter, r *http.Request) {
	if index().Render(context.Background(), w) != nil {
		return
	}
}

func main() {
	go hub.run()
	http.HandleFunc("/", handler)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
