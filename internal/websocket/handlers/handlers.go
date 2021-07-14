package handlers

import (
	"github.com/Mort4lis/ws-echo-server/internal/websocket"
	"log"
	"net/http"
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

		if err = conn.Send(websocket.Frame{
			Opcode:  typ,
			Payload: payload,
		}); err != nil {
			log.Println(err)
			return
		}
	}
}
