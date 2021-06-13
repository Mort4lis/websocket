package websocket

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
)

// GUID (Globally Unique Identifier)
const GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

var rawRespTemplate = "HTTP/1.1 101 Switching Protocols\n" +
	"Upgrade: websocket\n" +
	"Connection: Upgrade\n" +
	"Sec-WebSocket-Accept: %s\n\n"

type Websocket struct {
	conn    net.Conn
	buff    *bufio.ReadWriter
	headers http.Header
}

func NewWebsocket(w http.ResponseWriter, req *http.Request) (*Websocket, error) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("can't get control over tcp connection")
	}

	conn, buff, err := hj.Hijack()
	if err != nil {
		return nil, err
	}

	return &Websocket{
		conn:    conn,
		buff:    buff,
		headers: req.Header,
	}, nil
}

func (ws *Websocket) Handshake() error {
	secret := ws.createSecret(ws.headers.Get("Sec-WebSocket-Key"))
	rawResp := fmt.Sprintf(rawRespTemplate, secret)

	_, err := ws.conn.Write([]byte(rawResp))
	if err != nil {
		return err
	}
	return nil
}

func (ws *Websocket) createSecret(key string) string {
	hash := sha1.New()
	hash.Write([]byte(key))
	hash.Write([]byte(GUID))
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}
