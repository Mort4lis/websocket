package handlers

import (
	"github.com/Mort4lis/ws-echo-server/internal/websocket"
	"log"
	"net/http"
)

func initWebsocket(w http.ResponseWriter, req *http.Request) {
	var err error
	var frame websocket.Frame

	ws, err := websocket.NewWebsocket(w, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = ws.Handshake(); err != nil {
		log.Println(err)
		return
	}

	defer func() {
		if err == nil || !websocket.IsCloseError(err) {
			_ = ws.Close()
		}
	}()

	for {
		frame, err = ws.Receive()
		if err != nil {
			log.Println(err)
			return
		}

		if err = ws.Send(websocket.Frame{
			Opcode:  frame.Opcode,
			Payload: frame.Payload,
		}); err != nil {
			log.Println(err)
			return
		}
	}
}
