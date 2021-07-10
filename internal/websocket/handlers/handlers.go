package handlers

import (
	"github.com/Mort4lis/ws-echo-server/internal/websocket"
	"log"
	"net/http"
)

func initWebsocket(w http.ResponseWriter, req *http.Request) {
	var err error
	var frame websocket.Frame

	conn, err := websocket.Upgrade(w, req)
	if err != nil {
		return
	}

	defer func() {
		if err == nil || !websocket.IsCloseError(err) {
			_ = conn.Close()
		}
	}()

	for {
		frame, err = conn.Receive()
		if err != nil {
			log.Println(err)
			return
		}

		if err = conn.Send(websocket.Frame{
			Opcode:  frame.Opcode,
			Payload: frame.Payload,
		}); err != nil {
			log.Println(err)
			return
		}
	}
}
