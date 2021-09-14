package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Mort4lis/websocket"
)

func main() {
	dialer := &websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, err := dialer.Dial("ws://127.0.0.1:8080")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = conn.Close(); err != nil {
			log.Println(err)
		}
	}()

	sendStr := "hello, world!"
	if err = conn.WriteMessage(websocket.TextOpcode, []byte(sendStr)); err != nil {
		log.Fatal(err)
	}

	mt, payload, err := conn.ReadMessage()
	if err != nil {
		log.Fatal(err)
	}

	if mt != websocket.TextOpcode {
		log.Fatalf("expect text message, got %q", mt)
	}

	receivedStr := string(payload)
	fmt.Printf("sent message = %q, received = %q", sendStr, receivedStr)
}
