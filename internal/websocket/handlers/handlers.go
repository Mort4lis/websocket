package handlers

import (
	"fmt"
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
	}

	frame, err := ws.Receive()
	if err != nil {
		log.Println(err)
	}

	respBody := []byte(fmt.Sprintf("%s from Go echo server", frame.Payload))
	if err = ws.Send(websocket.Frame{
		Reserved: frame.Reserved,
		Opcode:   frame.Opcode,
		Payload:  respBody,
	}); err != nil {
		log.Println(err)
	}
}
