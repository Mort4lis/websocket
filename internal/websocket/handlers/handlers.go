package handlers

import (
	"github.com/Mort4lis/ws-echo-server/internal/websocket"
	"log"
	"net/http"
)

func initWebsocket(w http.ResponseWriter, req *http.Request) {
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
		_ = ws.Close()
	}()

	for {
		frame, err := ws.Receive()
		if err != nil {
			log.Println(err)
			return
		}

		switch frame.Opcode {
		case websocket.CloseOpcode:
			return
		default:
			if err = ws.Send(websocket.Frame{
				Opcode:  frame.Opcode,
				Payload: frame.Payload,
			}); err != nil {
				log.Println(err)
			}
		}
	}
}
