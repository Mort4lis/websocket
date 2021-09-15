package websocket

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

// Dialer is a type which represents the client settings to establish a
// WebSocket connection.
type Dialer struct {
	HandshakeTimeout time.Duration
	TLSConfig        *tls.Config

	wsKey string
}

// Dial creates a new client WebSocket connection using DialContext with a background context.
func (d *Dialer) Dial(urlStr string) (*Conn, error) {
	return d.DialContext(context.Background(), urlStr)
}

// DialContext creates a new client WebSocket connection.
//
// At first, it opens a new tcp connection over which it sends http handshake
// request for switching protocol to WebSocket. If handshake fails, DialContext returns
// HandshakeError with detailed reason about error.
func (d *Dialer) DialContext(ctx context.Context, urlStr string) (*Conn, error) {
	addr, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	switch addr.Scheme {
	case "ws":
		addr.Scheme = "http"
	case "wss":
		addr.Scheme = "https"
	default:
		return nil, fmt.Errorf("bad url schema (must be ws or wss)")
	}

	if d.HandshakeTimeout != 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, d.HandshakeTimeout)
		defer cancel()
	}

	wsKey, err := randomWebsocketKey()
	if err != nil {
		return nil, err
	}

	d.wsKey = wsKey

	req, err := d.prepareHandshakeRequest(ctx, addr.String())
	if err != nil {
		return nil, err
	}

	netConn, err := net.Dial("tcp", extractHostPort(addr))
	if err != nil {
		return nil, err
	}

	if addr.Scheme == "https" {
		tlsConn, err := d.tlsHandshake(netConn)
		if err != nil {
			return nil, err
		}

		netConn = tlsConn
	}

	if err = req.Write(netConn); err != nil {
		return nil, err
	}

	r := bufio.NewReader(netConn)
	if err = d.handleHandshakeResponse(r, req); err != nil {
		return nil, err
	}

	return &Conn{
		conn: netConn,
		rw:   bufio.NewReadWriter(r, bufio.NewWriter(netConn)),
	}, nil
}

func (d *Dialer) prepareHandshakeRequest(ctx context.Context, addr string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Upgrade", "WebSocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", d.wsKey)
	req.Header.Set("Sec-WebSocket-Version", "13")

	return req, nil
}

func (d *Dialer) handleHandshakeResponse(r *bufio.Reader, req *http.Request) error {
	resp, err := http.ReadResponse(r, req)
	if err != nil {
		return err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		return HandshakeError{"bad status code, expect status switching protocols (101)"}
	}

	if !checkHeaderContains(resp.Header, "Upgrade", "WebSocket") {
		return HandshakeError{"response Upgrade header value doesn't equal WebSocket"}
	}

	if !checkHeaderContains(resp.Header, "Connection", "Upgrade") {
		return HandshakeError{"response Connection header value doesn't equal Upgrade"}
	}

	if resp.Header.Get("Sec-Websocket-Accept") != hashWebsocketKey(d.wsKey) {
		return HandshakeError{"bad calculated Sec-Websocket-Accept header value"}
	}

	return nil
}

func (d *Dialer) tlsHandshake(netConn net.Conn) (*tls.Conn, error) {
	var tlsConfig *tls.Config
	if d.TLSConfig != nil {
		tlsConfig = d.TLSConfig.Clone()
	} else {
		tlsConfig = &tls.Config{}
	}

	tlsConn := tls.Client(netConn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return nil, err
	}

	return tlsConn, nil
}
