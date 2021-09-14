package websocket

import (
	"fmt"
	"net/http"
	"strings"
)

func newHandshakeError(w http.ResponseWriter, status int, reason string) error {
	err := HandshakeError{reason: reason}
	http.Error(w, err.Error(), status)

	return err
}

var handshakeResponseTemplate = strings.Join([]string{
	"HTTP/1.1 101 Switching Protocols",
	"Server: go/ws-custom-server",
	"Upgrade: WebSocket",
	"Connection: Upgrade",
	"Sec-WebSocket-Accept: %s",
	"", // required for extra CRLF
	"", // required for extra CRLF
}, "\r\n")

// Upgrade upgrades the HTTP connection protocol to WebSocket protocol.
func Upgrade(w http.ResponseWriter, req *http.Request) (*Conn, error) {
	if req.Method != http.MethodGet {
		return nil, newHandshakeError(w, http.StatusMethodNotAllowed, "request to upgrade is not GET")
	}

	if !checkHeaderContains(req.Header, "Connection", "Upgrade") {
		return nil, newHandshakeError(w, http.StatusBadRequest, "upgrade not found in Connection header")
	}

	if !checkHeaderContains(req.Header, "Upgrade", "WebSocket") {
		return nil, newHandshakeError(w, http.StatusBadRequest, "websocket not found in Upgrade header")
	}

	if !checkHeaderContains(req.Header, "Sec-WebSocket-Version", "13") {
		return nil, newHandshakeError(w, http.StatusBadRequest, "unsupported version for upgrade to websocket")
	}

	clientSecret := req.Header.Get("Sec-WebSocket-Key")
	if clientSecret == "" {
		return nil, newHandshakeError(w, http.StatusBadRequest, "Sec-Websocket-Key header is missing or blank")
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, newHandshakeError(w, http.StatusInternalServerError, "can't get control over tcp connection")
	}

	netConn, rw, err := hj.Hijack()
	if err != nil {
		return nil, newHandshakeError(w, http.StatusInternalServerError, err.Error())
	}

	rawResp := fmt.Sprintf(handshakeResponseTemplate, hashWebsocketKey(clientSecret))
	if _, err = netConn.Write([]byte(rawResp)); err != nil {
		_ = netConn.Close()

		return nil, err
	}

	return &Conn{
		isServer: true,
		conn:     netConn,
		rw:       rw,
	}, nil
}
