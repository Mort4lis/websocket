package main

import (
	"log"
	"net/http"

	"github.com/Mort4lis/websocket"
)

func initWebsocket(w http.ResponseWriter, req *http.Request) {
	conn, err := websocket.Upgrade(w, req)
	if err != nil {
		return
	}

	defer func() {
		_ = conn.Close()
	}()

	for {
		typ, payload, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)

			return
		}

		if err = conn.WriteMessage(typ, payload); err != nil {
			log.Println(err)

			return
		}
	}
}
