package websocket

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
)

// GUID (Globally Unique Identifier)
const GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

var handshakeRespTemplate = strings.Join([]string{
	"HTTP/1.1 101 Switching Protocols",
	"Server: go/ws-custom-server",
	"Upgrade: WebSocket",
	"Connection: Upgrade",
	"Sec-WebSocket-Accept: %s",
	"", // required for extra CRLF
	"", // required for extra CRLF
}, "\r\n")

type Websocket struct {
	conn    net.Conn
	rw      *bufio.ReadWriter
	headers http.Header
}

func NewWebsocket(w http.ResponseWriter, req *http.Request) (*Websocket, error) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("can't get control over tcp connection")
	}

	conn, rw, err := hj.Hijack()
	if err != nil {
		return nil, err
	}

	return &Websocket{
		conn:    conn,
		rw:      rw,
		headers: req.Header,
	}, nil
}

func (ws *Websocket) Handshake() error {
	secret := ws.createSecret(ws.headers.Get("Sec-WebSocket-Key"))
	rawResp := fmt.Sprintf(handshakeRespTemplate, secret)

	return ws.write([]byte(rawResp))
}

func (ws *Websocket) createSecret(key string) string {
	hash := sha1.New()
	hash.Write([]byte(key))
	hash.Write([]byte(GUID))
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}
